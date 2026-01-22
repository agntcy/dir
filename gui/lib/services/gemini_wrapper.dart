// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:google_generative_ai/google_generative_ai.dart';

abstract class GeminiFactory {
  GenerativeModelWrapper createModel({
    required String apiKey, 
    required String model, 
    List<Tool>? tools,
    String? systemInstruction,
  });
}

class RealGeminiFactory implements GeminiFactory {
  @override
  GenerativeModelWrapper createModel({
    required String apiKey, 
    required String model, 
    List<Tool>? tools,
    String? systemInstruction,
  }) {
    return RealGenerativeModel(GenerativeModel(
      model: model, 
      apiKey: apiKey, 
      tools: tools,
      systemInstruction: systemInstruction != null ? Content.text(systemInstruction) : null,
    ));
  }
}

abstract class GenerativeModelWrapper {
  ChatSessionWrapper startChat({List<Content>? history});
}

class RealGenerativeModel implements GenerativeModelWrapper {
  final GenerativeModel _model;
  RealGenerativeModel(this._model);

  @override
  ChatSessionWrapper startChat({List<Content>? history}) {
    return RealChatSession(_model.startChat(history: history));
  }
}

abstract class ChatSessionWrapper {
  Future<GenerateContentResponse> sendMessage(Content content);
  List<Content> get history;
}

class RealChatSession implements ChatSessionWrapper {
  final ChatSession _session;
  RealChatSession(this._session);

  @override
  Future<GenerateContentResponse> sendMessage(Content content) => _session.sendMessage(content);

  @override
  List<Content> get history => _session.history.toList();
}
