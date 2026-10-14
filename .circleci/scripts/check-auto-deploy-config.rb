#!/usr/bin/env ruby
# frozen_string_literal: true

# Pins the auto-deploy shape of .circleci/config_work.yml (VOIP-1586).
#
# Usage: check-auto-deploy-config.rb [config_work.yml] [auto-deploy-services.txt] [config.yml] [repo root] [auto-run-services.txt]
#
# Exit 0 when every rule holds, 1 with one line per violation otherwise.
# Parses the YAML (anchors and merge keys resolved), so comments and layout do
# not matter. Design: docs/plans/2026-10-13-main-auto-deploy-design.md (D5).

require 'yaml'

DEFAULT_CONFIG = File.expand_path('../config_work.yml', __dir__)
DEFAULT_LIST = File.expand_path('../auto-deploy-services.txt', __dir__)
DEFAULT_RUN_LIST = File.expand_path('../auto-run-services.txt', __dir__)
DEFAULT_SETUP = File.expand_path('../config.yml', __dir__)
DEFAULT_ROOT = File.expand_path('../..', __dir__)

# Services that must stay approval-kept. Hard-coded on purpose: the allowlist
# file is exactly what a careless change would edit.
DENY = %w[
  bin-trigger-sender voip-asterisk-proxy voip-rtpengine-proxy
  voip-kamailio-proxy
].freeze

ONLY_MAIN = { 'filters' => { 'branches' => { 'only' => 'main' } } }.freeze
IGNORE_MAIN = { 'filters' => { 'branches' => { 'ignore' => 'main' } } }.freeze

def load_yaml(path)
  # Psych 4+ (ruby >= 3.1) refuses aliases in load_file; ruby 3.0's Psych 3
  # has no unsafe_load_file and loads aliases by default.
  YAML.respond_to?(:unsafe_load_file) ? YAML.unsafe_load_file(path) : YAML.load_file(path)
end

# A workflow job entry is either "name" or { "key" => attrs }. Returns
# { key:, id:, attrs: } where id is the name the job is referenced by.
def normalize(entry)
  if entry.is_a?(Hash)
    key, attrs = entry.first
    attrs ||= {}
    { key: key, id: attrs['name'] || key, attrs: attrs }
  else
    { key: entry.to_s, id: entry.to_s, attrs: {} }
  end
end

def requires_of(job)
  Array(job[:attrs]['requires']).map(&:to_s)
end

def reaches?(jobs_by_id, from, target, seen = {})
  return false if seen[from]

  seen[from] = true
  job = jobs_by_id[from]
  return false unless job

  requires_of(job).any? { |r| r == target || reaches?(jobs_by_id, r, target, seen) }
end

# The shell command logical lines of a job's run steps (backslash-newline joined).
def logical_lines(job_def)
  Array(job_def && job_def['steps']).flat_map do |step|
    run = step.is_a?(Hash) ? step['run'] : nil
    # `- run: <string>` is the shorthand for `- run: { command: <string> }`.
    cmd = run.is_a?(Hash) ? run['command'] : run
    # Comment-only lines are not commands (a note may mention the script by name).
    cmd.to_s.gsub(/\\\n/, ' ').split("\n").reject { |l| l.strip.start_with?('#') }
  end
end

# The path-filtering mapping of the setup config as { pattern => parameter }.
def load_mapping(setup_path)
  setup = load_yaml(setup_path)
  filter = Array(setup.dig('workflows', 'check-changes', 'jobs')).map { |e| normalize(e) }
                      .find { |j| j[:key] == 'path-filtering/filter' }
  raise 'config.yml has no path-filtering/filter job in workflow check-changes' unless filter

  filter[:attrs]['mapping'].to_s.split("\n").map(&:strip).reject(&:empty?).map do |line|
    pattern, param, flag = line.split(/\s+/)
    raise "unparsable mapping line: #{line}" unless param && flag == 'true'

    [pattern, param]
  end.to_h
end

PARAM_REF = /\A<<\s*pipeline\.parameters\.([A-Za-z0-9_-]+)\s*>>\z/.freeze

# The paths whose newest commit decides whether a deploy is stale: the service
# directory first, then the directory of every extra run-bin-X parameter that
# arms the workflow (bin-api-manager is armed by bin-openapi-manager too).
# Returns [paths, error] with paths nil on error.
def guard_paths_for(svc, wf, mapping_loader, repo_root)
  w = wf['when']
  refs = case w
         when String then [w]
         when Hash then w.keys == ['or'] && w['or'].is_a?(Array) ? w['or'] : nil
         end
  params = refs && refs.map { |r| r.is_a?(String) && (m = PARAM_REF.match(r)) ? m[1] : nil }
  return [nil, "workflow #{svc}: unsupported when form (only a run-X parameter or or: [run-X, ...] is allowed)"] if params.nil? || params.any?(&:nil?)

  own = "run-#{svc}"
  return [nil, "workflow #{svc}: when must contain its own parameter #{own}"] unless params.include?(own)

  paths = [svc]
  (params - [own]).each do |param|
    dir = param.sub(/\Arun-/, '')
    mapping = mapping_loader.call
    unless param.start_with?('run-bin-') && mapping["#{dir}/.*"] == param && File.directory?(File.join(repo_root, dir))
      return [nil, "workflow #{svc}: when references #{param} that is not a mapped service directory"]
    end
    paths << dir
  end
  [paths, nil]
end

def check(config_path, list_path, setup_path = DEFAULT_SETUP, repo_root = DEFAULT_ROOT, run_list_path = DEFAULT_RUN_LIST)
  errors = []
  cfg = load_yaml(config_path)
  mapping = nil
  mapping_loader = -> { mapping ||= load_mapping(setup_path) }
  list_lines = File.readlines(list_path).map { |l| l.sub(/#.*/, '').strip }.reject(&:empty?)
  run_lines = File.readlines(run_list_path).map { |l| l.sub(/#.*/, '').strip }.reject(&:empty?)
  workflows = cfg['workflows'] || {}
  jobs = cfg['jobs'] || {}

  # (c) the two filter anchors are exactly what the shape relies on.
  errors << 'deploy_filters anchor is not exactly only: main' unless cfg['deploy_filters'] == ONLY_MAIN
  errors << 'non_main_filters anchor is not exactly ignore: main' unless cfg['non_main_filters'] == IGNORE_MAIN

  # (d)/(f) allowlist hygiene.
  dups = list_lines.group_by { |x| x }.select { |_, v| v.size > 1 }.keys
  dups.each { |d| errors << "allowlist has a duplicate entry: #{d}" }
  list_lines.each do |svc|
    errors << "allowlist entry is not allowed to be auto-deployed: #{svc}" if DENY.include?(svc)
    errors << "allowlist entry has no workflow in config: #{svc}" unless workflows.key?(svc)
  end

  # Verify-only (auto-run) list hygiene: disjoint from the deploy allowlist and
  # from the deny list, no duplicates, every entry has a workflow.
  run_dups = run_lines.group_by { |x| x }.select { |_, v| v.size > 1 }.keys
  run_dups.each { |d| errors << "auto-run list has a duplicate entry: #{d}" }
  run_lines.each do |svc|
    errors << "auto-run list entry is not allowed to be auto-run: #{svc}" if DENY.include?(svc)
    errors << "auto-run list entry is also in the auto-deploy allowlist: #{svc}" if list_lines.include?(svc)
    errors << "auto-run list entry has no workflow in config: #{svc}" unless workflows.key?(svc)
  end

  workflows.each do |name, wf|
    unless wf.is_a?(Hash) && wf['jobs'].is_a?(Array)
      errors << "workflow #{name} is in the allowlist but has no jobs list" if list_lines.include?(name) || run_lines.include?(name)
      next
    end

    js = wf['jobs'].map { |e| normalize(e) }
    has_auto = js.any? { |j| j[:id].end_with?('-auto') }
    auto_kind = js.any? { |j| j[:id].end_with?('-test-auto') } ? '-test-auto' : '-auto'
    listed = list_lines.include?(name)
    listed_run = run_lines.include?(name)

    # (d) shape without allowlist entry, and (b) none for everyone else.
    errors << "workflow #{name} has a #{auto_kind} job but is not in the allowlist" if has_auto && !listed && !listed_run

    if listed && listed_run
      next # already reported as a list conflict above
    elsif listed_run
      check_autorun_workflow(name, js, jobs, errors, cfg['commands'] || {})
    elsif listed
      paths, err = guard_paths_for(name, wf, mapping_loader, repo_root)
      errors << err if err
      check_auto_workflow(name, js, jobs, errors, paths)
    else
      check_kept_workflow(name, js, jobs, errors)
    end
  end
  errors
end

# (a) the exact auto-deploy shape.
def check_auto_workflow(svc, js, jobs, errors, paths)
  where = "workflow #{svc}"
  expected = ['build-approval', "#{svc}-test", "#{svc}-test-auto", "#{svc}-build", "#{svc}-deploy"]
  ids = js.map { |j| j[:id] }
  errors << "#{where}: jobs must be exactly #{expected.join(', ')} (got #{ids.join(', ')})" unless ids.sort == expected.sort

  by_id = js.to_h { |j| [j[:id], j] }
  approval = by_id['build-approval']
  if approval
    a = approval[:attrs]
    errors << "#{where}: build-approval must be type: approval" unless a['type'] == 'approval'
    errors << "#{where}: build-approval filters must be exactly ignore: main" unless a['filters'] == IGNORE_MAIN['filters']
  end

  test = by_id["#{svc}-test"]
  if test
    errors << "#{where}: #{svc}-test filters must be exactly ignore: main" unless test[:attrs]['filters'] == IGNORE_MAIN['filters']
    errors << "#{where}: #{svc}-test must require exactly build-approval" unless requires_of(test) == ['build-approval']
  end

  auto = by_id["#{svc}-test-auto"]
  if auto
    errors << "#{where}: #{svc}-test-auto key must be #{svc}-test" unless auto[:key] == "#{svc}-test"
    errors << "#{where}: #{svc}-test-auto filters must be exactly only: main" unless auto[:attrs]['filters'] == ONLY_MAIN['filters']
    errors << "#{where}: #{svc}-test-auto must not have requires" if auto[:attrs].key?('requires')
  end

  build = by_id["#{svc}-build"]
  if build && requires_of(build).sort != ["#{svc}-test", "#{svc}-test-auto"].sort
    errors << "#{where}: #{svc}-build must require exactly #{svc}-test and #{svc}-test-auto"
  end

  deploy = by_id["#{svc}-deploy"]
  return unless deploy

  d = deploy[:attrs]
  errors << "#{where}: #{svc}-deploy key must be #{svc}-deploy (it selects the job definition)" unless deploy[:key] == "#{svc}-deploy"
  errors << "#{where}: #{svc}-deploy filters must be exactly only: main" unless d['filters'] == ONLY_MAIN['filters']
  errors << "#{where}: #{svc}-deploy must require exactly #{svc}-build" unless requires_of(deploy) == ["#{svc}-build"]
  want_group = "<< pipeline.project.slug >>/deploy-#{svc}"
  errors << "#{where}: #{svc}-deploy serial-group must be #{want_group}" unless d['serial-group'] == want_group
  errors << "#{where}: #{svc}-deploy must use the production context" unless Array(d['context']) == ['production']

  # The guard env must sit on the very command that runs komodo-api-deploy.sh,
  # with this service's path and this service's stack name.
  lines = logical_lines(jobs["#{svc}-deploy"]).select { |l| l.include?('komodo-api-deploy.sh') }
  if lines.size != 1
    errors << "#{where}: job #{svc}-deploy must run komodo-api-deploy.sh exactly once (found #{lines.size})"
  else
    # Leading VAR=value assignments, then the script path, then its arguments.
    # Anything else in front of the script (a comment, `env -u`, `true;`) fails.
    m = lines.first.match(/\A\s*((?:[A-Za-z_][A-Za-z0-9_]*=(?:"[^"]*"|'[^']*'|\S*)\s+)*)(\S*komodo-api-deploy\.sh)\s+(\S+)/)
    if m.nil?
      errors << "#{where}: #{svc}-deploy command must be VAR=value assignments followed by komodo-api-deploy.sh"
    else
      # Tokenize the assignments so only a variable whose whole name matches counts.
      assigns = m[1].scan(/([A-Za-z_][A-Za-z0-9_]*)=(?:"([^"]*)"|'([^']*)'|(\S*))/)
      guards = assigns.select { |name, *_| name == 'DEPLOY_REQUIRE_LATEST_FOR_PATHS' }.map { |_, *vals| vals.compact.first.to_s }
      want = (paths || [svc]).join(' ')
      unless guards == [want]
        errors << %(#{where}: #{svc}-deploy command must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="#{want}" exactly once on the same command as komodo-api-deploy.sh)
      end
      errors << "#{where}: komodo-api-deploy.sh first argument must be #{svc}" unless m[3] == svc
    end
  end
end

# The exact auto-run shape (VOIP-1597): a verify-only workflow whose job runs on
# main without approval. No deploy, no build, no production context, no push.
AUTORUN_STEPS = %w[checkout run go-test store_test_results].freeze
AUTORUN_FORBIDDEN = /komodo-api-deploy\.sh|docker\b[^\n]*\b(push|login|build|tag)\b|docker-(push|build)|buildx|kubectl|skopeo|crane|git\s+push|gcloud/.freeze

# Command lines of every run step, including steps nested in when/unless.
def deep_run_lines(steps)
  Array(steps).flat_map do |step|
    next [] unless step.is_a?(Hash)

    nested = step.values.flat_map { |v| v.is_a?(Hash) ? deep_run_lines(v['steps']) : [] }
    nested + logical_lines('steps' => [step])
  end
end

def check_autorun_workflow(svc, js, jobs, errors, commands = {})
  where = "workflow #{svc}"
  ids = js.map { |j| j[:id] }
  by_id = js.to_h { |j| [j[:id], j] }
  autos = ids.select { |i| i.end_with?('-auto') }
  bases = ids - ['build-approval'] - autos
  unless ids.size == 3 && by_id['build-approval'] && autos.size == 1 && bases.size == 1
    errors << "#{where}: jobs must be exactly build-approval, one verify job and its -auto copy (got #{ids.join(', ')})"
    return
  end

  # The allowed go-test command is scanned like a run step (same forbidden list).
  gt = commands['go-test']
  if gt.nil? || deep_run_lines(gt['steps']).any? { |l| AUTORUN_FORBIDDEN.match?(l) }
    errors << "#{where}: command go-test is missing or runs a deploy, push, login, build, tag or gcloud"
  end

  base = bases.first
  errors << "#{where}: verify job #{base} must be named #{svc}-test or #{svc}-validate" unless ["#{svc}-test", "#{svc}-validate"].include?(base)

  a = by_id['build-approval'][:attrs]
  errors << "#{where}: build-approval must be type: approval" unless a['type'] == 'approval'
  errors << "#{where}: build-approval filters must be exactly ignore: main" unless a['filters'] == IGNORE_MAIN['filters']

  b = by_id[base]
  errors << "#{where}: #{base} key must be #{base}" unless b[:key] == base
  errors << "#{where}: #{base} filters must be exactly ignore: main" unless b[:attrs]['filters'] == IGNORE_MAIN['filters']
  errors << "#{where}: #{base} must require exactly build-approval" unless requires_of(b) == ['build-approval']

  auto = by_id["#{base}-auto"]
  if auto.nil?
    errors << "#{where}: the -auto copy must be named #{base}-auto"
  else
    errors << "#{where}: #{base}-auto key must be #{base}" unless auto[:key] == base
    errors << "#{where}: #{base}-auto filters must be exactly only: main" unless auto[:attrs]['filters'] == ONLY_MAIN['filters']
    errors << "#{where}: #{base}-auto must not have requires" if auto[:attrs].key?('requires')
  end

  # Nothing that can reach production: no context, no deploy or build job, no
  # deploy or push command in the job definitions. (Commands hidden inside a
  # reusable command are not scanned; review covers those.)
  js.each do |j|
    errors << "#{where}: job #{j[:id]} must not use a context" if j[:attrs].key?('context')
    if j[:id].end_with?('-deploy') || j[:key].end_with?('-deploy') || j[:id].end_with?('-build') || j[:key].end_with?('-build')
      errors << "#{where}: job #{j[:id]} must not be a build or deploy job"
    end
    # Steps allowlist: any other command call (docker-build, setup_remote_docker,
    # an orb command, ...) could hide a build, push or deploy, so it is rejected.
    Array(jobs[j[:key]] && jobs[j[:key]]['steps']).each do |step|
      name = step.is_a?(Hash) ? step.keys.first : step.to_s
      next if AUTORUN_STEPS.include?(name)

      errors << "#{where}: job definition #{j[:key]} uses a step that is not allowed in a verify-only job: #{name}"
    end
    if logical_lines(jobs[j[:key]]).any? { |l| AUTORUN_FORBIDDEN.match?(l) }
      errors << "#{where}: job definition #{j[:key]} must not run a deploy, a docker push, login or build, or gcloud"
    end
  end
end

# (b)/(e) workflows that must keep approval before deploy.
def check_kept_workflow(name, js, jobs, errors)
  where = "workflow #{name}"
  by_id = js.to_h { |j| [j[:id], j] }
  # A deploy job is one named *-deploy (by key or by name) or whose job
  # definition runs komodo-api-deploy.sh, so renaming it does not hide it.
  deploys = js.select do |j|
    j[:id].end_with?('-deploy') || j[:key].end_with?('-deploy') ||
      logical_lines(jobs[j[:key]]).any? { |l| l.include?('komodo-api-deploy.sh') }
  end
  # The jobs a deploy depends on (test, build, ...) sit between the approval and
  # the deploy; a branch filter on any of them could skip it on main and let the
  # pipeline run without the approval. Jobs outside that chain (for example the
  # main-only migration checkpoint of bin-dbscheme-manager) are not constrained.
  chain = {}
  collect = lambda do |id|
    requires_of(by_id[id] || { attrs: {} }).each do |r|
      next if chain[r]

      chain[r] = true
      collect.call(r)
    end
  end
  deploys.each { |dj| collect.call(dj[:id]) }

  js.each do |j|
    filters = j[:attrs]['filters']
    if filters.is_a?(Hash) && filters.dig('branches', 'ignore')
      errors << "#{where}: job #{j[:id]} has an ignore branch filter but #{name} is not auto-deploy"
    end
    if deploys.include?(j)
      # The only branch filter an approval-kept deploy may carry is deploy_filters.
      if !filters.nil? && filters != ONLY_MAIN['filters']
        errors << "#{where}: deploy job #{j[:id]} has a branch filter other than only: main"
      end
    elsif chain[j[:id]] && j[:id] != 'build-approval' && !filters.nil?
      errors << "#{where}: job #{j[:id]} has a branch filter but #{name} is not auto-deploy"
    end
  end
  return if deploys.empty?

  approval = by_id['build-approval']
  if approval.nil? || approval[:attrs]['type'] != 'approval' || approval[:attrs].key?('filters')
    errors << "#{where}: approval-kept workflow needs an unfiltered build-approval of type approval"
    return
  end
  deploys.each do |dj|
    unless reaches?(by_id, dj[:id], 'build-approval')
      errors << "#{where}: #{dj[:id]} is not gated by build-approval through its requires chain"
    end
  end
end

if $PROGRAM_NAME == __FILE__
  config_path = ARGV[0] || DEFAULT_CONFIG
  list_path = ARGV[1] || DEFAULT_LIST
  setup_path = ARGV[2] || DEFAULT_SETUP
  repo_root = ARGV[3] || DEFAULT_ROOT
  run_list_path = ARGV[4] || DEFAULT_RUN_LIST
  begin
    errors = check(config_path, list_path, setup_path, repo_root, run_list_path)
  rescue StandardError => e
    warn "check-auto-deploy-config: cannot check: #{e.class}: #{e.message}"
    exit 1
  end
  if errors.empty?
    puts 'check-auto-deploy-config: OK'
    exit 0
  end
  errors.each { |e| warn "check-auto-deploy-config: #{e}" }
  exit 1
end
