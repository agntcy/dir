// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/model.dart';

void main() {
  group('McpTool Tests', () {
    test('fromJson parses correctly', () {
      final json = {
        'name': 'tool1',
        'description': 'desc1',
        'inputSchema': {'type': 'object'}
      };
      final tool = McpTool.fromJson(json);
      expect(tool.name, 'tool1');
      expect(tool.description, 'desc1');
      expect(tool.inputSchema, {'type': 'object'});
    });

    test('fromJson handles defaults', () {
      final json = {
        'name': 'tool1',
      };
      final tool = McpTool.fromJson(json);
      expect(tool.description, '');
      expect(tool.inputSchema, {});
    });
  });

  group('McpPrompt Tests', () {
    test('fromJson parses correctly', () {
      final json = {
        'name': 'prompt1',
        'description': 'desc1',
        'arguments': [
          {'name': 'arg1', 'description': 'argDesc', 'required': true}
        ]
      };
      final prompt = McpPrompt.fromJson(json);
      expect(prompt.name, 'prompt1');
      expect(prompt.arguments.first.name, 'arg1');
      expect(prompt.arguments.first.required, true);
    });
  });

  group('McpGetPromptResult Tests', () {
    test('fromJson parses correctly', () {
      final json = {
        'description': 'Result desc',
        'messages': [
          {
            'role': 'user',
            'content': {
              'type': 'text',
              'text': 'Hello'
            }
          }
        ]
      };
      final result = McpGetPromptResult.fromJson(json);
      expect(result.description, 'Result desc');
      expect(result.messages.length, 1);
      expect(result.messages.first.role, 'user');
      expect(result.messages.first.content.text, 'Hello');
    });

     test('fromJson handles missing messages', () {
      final json = {
        'description': 'Result desc',
      };
      final result = McpGetPromptResult.fromJson(json);
      expect(result.description, 'Result desc');
      expect(result.messages, isEmpty);
    });
  });

  group('McpToolResult Tests', () {
    test('stores content and error state', () {
       final result = McpToolResult(content: 'test', isError: true);
       expect(result.content, 'test');
       expect(result.isError, true);
    });

    test('defaults isError to false', () {
       final result = McpToolResult(content: 'test');
       expect(result.isError, false);
    });
  });

  group('McpToolCall Tests', () {
    test('stores values correctly', () {
      final call = McpToolCall(name: 'tool', arguments: {'a': 1});
      expect(call.name, 'tool');
      expect(call.arguments['a'], 1);
    });
  });
}
