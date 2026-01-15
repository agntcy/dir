// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/client.dart';

void main() {
  test('Integration: MCP Client connects to local binary', () async {
    // We are running from 'gui/' directory usually.
    // The binary is at '../bin' relative to 'gui/'.
    // If running via 'flutter test', the CWD is usually the project root (gui/).
    final String binaryPath = '../bin/mcp-server';

    final file = File(binaryPath);
    if (!file.existsSync()) {
      // Try absolute path if relative fails
      final absPath = File('${Directory.current.parent.path}/bin/mcp-server');
      if (!absPath.existsSync()) {
        print("CWD: ${Directory.current.path}");
        fail('MCP binary not found at $binaryPath or ${absPath.path}. Please build the server first.');
      }
    }

    print('Using binary at: $binaryPath');

    final client = McpClient(executablePath: binaryPath);

    // 1. Start the process
    // Pass DISABLE env to avoid log noise breaking the stdio protocol during tests
    await client.start(environment: {
      "OASF_API_VALIDATION_DISABLE": "true",
    });

    // 2. Initialize
    print('Initializing...');
    await client.initialize();
    print('Initialized.');

    // 3. List Tools
    print('Listing tools...');
    final tools = await client.listTools();

    final toolNames = tools.map((t) => t.name).toList();
    print('Found tools: $toolNames');

    // 4. Verify specific tool exists
    expect(toolNames, contains('agntcy_dir_search_local'));

    // 5. Test Tool Call (Search)
    // We expect it to return an error because we don't pass filters,
    // exactly matching the manual python test result.
    print('Calling search tool...');
    final result = await client.callTool('agntcy_dir_search_local', {"limit": 1});
    print('Result content: ${result.content}');

    // The previous manual test result was:
    // {"count":0,"error_message":"at least one query filter must be provided","has_more":false}
    // embedded in a text content block.

    // Check if we got a response (even if it's the error message from the tool logic)
    // The result.content is usually a List.
    final contentList = result.content as List;
    expect(contentList, isNotEmpty);
    final textObj = contentList.first as Map<String, dynamic>;
    expect(textObj['type'], 'text');
    expect(textObj['text'], contains('at least one query filter must be provided'));

    await client.stop();
  });
}
