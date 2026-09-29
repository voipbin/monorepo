# GitHub MCP schema fixtures

The `*.json` files in this directory (except `incident_x_mcp_header.json` and
the `*.golden.json` files) are the `inputSchema` objects extracted from GitHub
MCP server tool snapshots, `pkg/github/__toolsnaps__/<tool>.snap`, at upstream
commit `85598ba` of https://github.com/github/github-mcp-server:

- `issue_write.json`
- `projects_write.json`
- `update_issue_labels.json`
- `custom_properties_write.json`
- `get_file_contents.json`

`incident_x_mcp_header.json` is synthetic. It reproduces the production-only
`x-mcp-header` shape on the `owner`/`repo` properties, which the upstream
snapshots do not carry.

The `*.golden.json` files are the output of `mcpschema.Normalize` for each
fixture, generated with `go test ./pkg/mcpschema/ -run Fixtures -update` and
reviewed by hand against the design
(`docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md`, section 15.7).

The upstream files are used under the MIT License:

```
MIT License

Copyright (c) 2025 GitHub

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
