// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/llm_provider.dart';
import 'package:gui/services/gemini_wrapper.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:http/http.dart' as http;

// Mock Classes
class MockGeminiFactory implements GeminiFactory {
  MockGenerativeModelWrapper? lastCreatedModel;
  List<Tool>? lastTools;

  @override
  GenerativeModelWrapper createModel({
    required String apiKey,
    required String model,
    List<Tool>? tools,
    String? systemInstruction,
  }) {
    lastTools = tools;
    lastCreatedModel = MockGenerativeModelWrapper();
    return lastCreatedModel!;
  }
}

class MockGenerativeModelWrapper implements GenerativeModelWrapper {
  @override
  ChatSessionWrapper startChat({List<Content>? history}) {
    return MockChatSessionWrapper();
  }
}

class MockChatSessionWrapper implements ChatSessionWrapper {
  @override
  List<Content> get history => [];

  @override
  Future<GenerateContentResponse> sendMessage(Content content) async {
    return GenerateContentResponse([], null);
  }
}

class MockHttpClient extends Fake implements http.Client {
  final Future<http.Response> Function(Uri url, {Map<String, String>? headers, Object? body, Encoding? encoding}) _postHandler;

  MockHttpClient(this._postHandler);

  @override
  Future<http.Response> post(Uri url, {Map<String, String>? headers, Object? body, Encoding? encoding}) {
    return _postHandler(url, headers: headers, body: body, encoding: encoding);
  }
}

void main() {
  group('LlmProvider - Gemini Tests', () {
    test('GeminiProvider init converts tools correctly', () async {
      final factory = MockGeminiFactory();
      final provider = GeminiProvider(apiKey: 'key', geminiFactory: factory);

      final tools = [
        McpTool(
          name: 'tool1',
          description: 'desc1',
          inputSchema: {
            'type': 'object',
            'properties': {
              'arg1': {'type': 'string'}
            }
          }
        )
      ];

      await provider.init(tools);

      expect(factory.lastTools, isNotNull);
      expect(factory.lastTools!.length, 1);
      final func = factory.lastTools!.first.functionDeclarations!.first;
      expect(func.name, 'tool1');
      expect(func.description, 'desc1');
      expect(func.parameters!.properties!['arg1']!.type, SchemaType.string);
    });
  });

  group('LlmProvider - OpenAI/Azure Tests', () {
    test('OpenAiCompatibleProvider makes correct request', () async {
      final mockClient = MockHttpClient((url, {headers, body, encoding}) async {
        expect(url.toString(), 'https://api.example.com/v1/chat/completions');
        expect(headers!['Authorization'], 'Bearer sk-test');
        expect(headers['Content-Type'], 'application/json');

        final jsonBody = jsonDecode(body as String);
        expect(jsonBody['model'], 'gpt-4o');
        expect(jsonBody['messages'][0]['role'], 'user');
        expect(jsonBody['messages'][0]['content'], 'Hello');

        return http.Response(jsonEncode({
          'choices': [
            {
              'message': {
                'content': 'Response text',
                'tool_calls': []
              }
            }
          ]
        }), 200);
      });

      final provider = OpenAiCompatibleProvider(
        apiKey: 'sk-test',
        endpoint: 'https://api.example.com/v1',
        client: mockClient,
      );

      await provider.init([]);
      final response = await provider.sendMessage('Hello', []);
      expect(response.text, 'Response text');
    });

    test('AzureOpenAiProvider makes correct request', () async {
      final mockClient = MockHttpClient((url, {headers, body, encoding}) async {
        expect(url.toString(), 'https://my-resource.openai.azure.com/openai/deployments/dep-1/chat/completions?api-version=2024-10-21');
        expect(headers!['api-key'], 'azure-key');

        final jsonBody = jsonDecode(body as String);
        expect(jsonBody['messages'][0]['role'], 'user');
        expect(jsonBody['messages'][0]['content'], 'Hello');

        return http.Response(jsonEncode({
          'choices': [
             {
              'message': {
                'content': 'Azure response',
                'tool_calls': [
                  {
                    'id': 'call_1',
                    'function': {
                      'name': 'tool1',
                      'arguments': '{"arg": "val"}'
                    }
                  }
                ]
              }
            }
          ]
        }), 200);
      });

      final provider = AzureOpenAiProvider(
        apiKey: 'azure-key',
        endpoint: 'my-resource.openai.azure.com',
        deploymentId: 'dep-1',
        client: mockClient,
      );

      await provider.init([]);
      final response = await provider.sendMessage('Hello', []);
      expect(response.text, 'Azure response');
      expect(response.toolCalls.length, 1);
      expect(response.toolCalls.first.name, 'tool1');
    });
  });

  group('LlmResponse Tests', () {
    test('stores data', () {
      final resp = LlmResponse(text: 'text', toolCalls: [LlmToolCall('name', {})]);
      expect(resp.text, 'text');
      expect(resp.toolCalls.length, 1);
    });
  });
}
