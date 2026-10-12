#!/usr/bin/env ruby
# frozen_string_literal: true

# Pins the auto-deploy shape of .circleci/config_work.yml (VOIP-1586).
#
# Usage: check-auto-deploy-config.rb [config_work.yml] [auto-deploy-services.txt]
#
# Exit 0 when every rule holds, 1 with one line per violation otherwise.
# Parses the YAML (anchors and merge keys resolved), so comments and layout do
# not matter. Design: docs/plans/2026-10-13-main-auto-deploy-design.md (D5).

require 'yaml'

DEFAULT_CONFIG = File.expand_path('../config_work.yml', __dir__)
DEFAULT_LIST = File.expand_path('../auto-deploy-services.txt', __dir__)

# Services that must stay approval-kept. Hard-coded on purpose: the allowlist
# file is exactly what a careless change would edit.
DENY = %w[
  bin-timeline-manager bin-rag-manager bin-trigger-sender
  voip-asterisk-proxy voip-rtpengine-proxy voip-kamailio-proxy
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
    cmd.to_s.gsub(/\\\n/, ' ').split("\n")
  end
end

def check(config_path, list_path)
  errors = []
  cfg = load_yaml(config_path)
  list_lines = File.readlines(list_path).map { |l| l.sub(/#.*/, '').strip }.reject(&:empty?)
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

  workflows.each do |name, wf|
    unless wf.is_a?(Hash) && wf['jobs'].is_a?(Array)
      errors << "workflow #{name} is in the allowlist but has no jobs list" if list_lines.include?(name)
      next
    end

    js = wf['jobs'].map { |e| normalize(e) }
    has_auto = js.any? { |j| j[:id].end_with?('-test-auto') }
    listed = list_lines.include?(name)

    # (d) shape without allowlist entry, and (b) none for everyone else.
    errors << "workflow #{name} has a -test-auto job but is not in the allowlist" if has_auto && !listed

    if listed
      check_auto_workflow(name, js, jobs, errors)
    else
      check_kept_workflow(name, js, errors)
    end
  end
  errors
end

# (a) the exact auto-deploy shape.
def check_auto_workflow(svc, js, jobs, errors)
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
    line = lines.first
    unless line.include?(%(DEPLOY_REQUIRE_LATEST_FOR_PATHS="#{svc}"))
      errors << %(#{where}: #{svc}-deploy command must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="#{svc}" on the same command as komodo-api-deploy.sh)
    end
    unless line =~ %r{komodo-api-deploy\.sh\s+#{Regexp.escape(svc)}\s}
      errors << "#{where}: komodo-api-deploy.sh first argument must be #{svc}"
    end
  end
end

# (b)/(e) workflows that must keep approval before deploy.
def check_kept_workflow(name, js, errors)
  where = "workflow #{name}"
  by_id = js.to_h { |j| [j[:id], j] }
  js.each do |j|
    if j[:attrs]['filters'].is_a?(Hash) && j[:attrs]['filters'].dig('branches', 'ignore')
      errors << "#{where}: job #{j[:id]} has an ignore branch filter but #{name} is not auto-deploy"
    end
  end
  deploys = js.select { |j| j[:id].end_with?('-deploy') }
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
  begin
    errors = check(config_path, list_path)
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
