// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


class McpTool {
  final String name;
  final String description;
  final Map<String, dynamic> inputSchema;

  McpTool({
    required this.name,
    required this.description,
    required this.inputSchema,
  });

  factory McpTool.fromJson(Map<String, dynamic> json) {
    return McpTool(
      name: json['name'] as String,
      description: json['description'] as String? ?? '',
      inputSchema: json['inputSchema'] as Map<String, dynamic>? ?? {},
    );
  }
}

class McpToolCall {
  final String name;
  final Map<String, dynamic> arguments;

  McpToolCall({required this.name, required this.arguments});
}

class McpToolResult {
  final dynamic content;
  final bool isError;

  McpToolResult({required this.content, this.isError = false});
}

class McpPrompt {
  final String name;
  final String description;
  final List<McpPromptArgument> arguments;

  McpPrompt({
    required this.name,
    required this.description,
    required this.arguments,
  });

  factory McpPrompt.fromJson(Map<String, dynamic> json) {
    var argsList = <McpPromptArgument>[];
    if (json['arguments'] != null) {
      argsList = (json['arguments'] as List)
          .map((a) => McpPromptArgument.fromJson(a))
          .toList();
    }
    return McpPrompt(
      name: json['name'] as String,
      description: json['description'] as String? ?? '',
      arguments: argsList,
    );
  }
}

class McpPromptArgument {
  final String name;
  final String description;
  final bool required;

  McpPromptArgument({
    required this.name,
    required this.description,
    required this.required,
  });

  factory McpPromptArgument.fromJson(Map<String, dynamic> json) {
    return McpPromptArgument(
      name: json['name'] as String,
      description: json['description'] as String? ?? '',
      required: json['required'] as bool? ?? false,
    );
  }
}

class McpGetPromptResult {
  final String description;
  final List<McpPromptMessage> messages;

  McpGetPromptResult({required this.description, required this.messages});

  factory McpGetPromptResult.fromJson(Map<String, dynamic> json) {
    var msgs = <McpPromptMessage>[];
    if (json['messages'] != null) {
      msgs = (json['messages'] as List)
          .map((m) => McpPromptMessage.fromJson(m))
          .toList();
    }
    return McpGetPromptResult(
      description: json['description'] as String? ?? '',
      messages: msgs,
    );
  }
}

class McpPromptMessage {
  final String role;
  final McpContent content;

  McpPromptMessage({required this.role, required this.content});

  factory McpPromptMessage.fromJson(Map<String, dynamic> json) {
    return McpPromptMessage(
      role: json['role'] as String,
      content: McpContent.fromJson(json['content']),
    );
  }
}

class McpContent {
  final String type;
  final String text; // simplified

  McpContent({required this.type, required this.text});

  factory McpContent.fromJson(Map<String, dynamic> json) {
    return McpContent(
      type: json['type'] as String? ?? 'text',
      text: json['text'] as String? ?? '',
    );
  }
}
