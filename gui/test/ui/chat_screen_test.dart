// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';
import 'package:gui/ui/chat_screen.dart';

class MockAiService implements AiService {
  @override
  McpClient? mcpClient;

  @override
  Future<void> init(LlmProvider provider) async {}

  @override
  Future<String?> sendMessage(
    String message,
    List<Content> history, {
    void Function(String, dynamic)? onToolOutput,
  }) async {
    return 'echo: $message';
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  testWidgets('ChatScreen sends message and shows response', (WidgetTester tester) async {
    final mockService = MockAiService();

    await tester.pumpWidget(MaterialApp(
      home: ChatScreen(aiService: mockService),
    ));

    // Wait for init (sync in our mock)
    await tester.pump();

    // Enter text
    await tester.enterText(find.byType(TextField), 'Hello');
    await tester.tap(find.byIcon(Icons.send));

    // Rebuild
    await tester.pump();
    // Wait for future
    await tester.pump(const Duration(milliseconds: 100)); // drain microtasks

    // Check user message
    expect(find.text('Hello'), findsOneWidget);

    // Check model response (MarkdownBody renders text)
    expect(find.text('echo: Hello'), findsOneWidget);
  });
}
