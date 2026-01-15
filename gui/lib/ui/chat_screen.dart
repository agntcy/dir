// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import '../main.dart';
import '../mcp/client.dart';
import '../services/ai_service.dart';
import '../services/llm_provider.dart';
import 'widgets/record_card.dart';

class ChatScreen extends StatefulWidget {
  final AiService? aiService;

  const ChatScreen({super.key, this.aiService});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _controller = TextEditingController();
  final List<Content> _history = [];
  final List<Map<String, dynamic>> _messages = []; // For UI display

  AiService? _aiService;
  McpClient? _mcpClient;
  bool _isLoading = false;

  // Config
  String _providerType = 'gemini'; // gemini, azure
  String? _apiKey;
  String? _azureEndpoint;
  String? _azureDeployment;
  String _azureApiVersion = '2024-10-21'; // Default

  @override
  void initState() {
    super.initState();
    if (widget.aiService != null) {
      _aiService = widget.aiService;
    } else {
      _checkEnvAndInit();
    }
  }

  void _checkEnvAndInit() {
    // Check for Azure Env (Runtime env preferred, then compile-time)
    final azureKey = Platform.environment['AZURE_API_KEY'] ?? const String.fromEnvironment('AZURE_API_KEY');
    final azureEp = Platform.environment['AZURE_ENDPOINT'] ?? const String.fromEnvironment('AZURE_ENDPOINT');
    final azureDep = Platform.environment['AZURE_DEPLOYMENT'] ?? const String.fromEnvironment('AZURE_DEPLOYMENT');

    if (azureKey.isNotEmpty && azureEp.isNotEmpty && azureDep.isNotEmpty) {
        print('Auto-configuring Azure from Environment');
        _providerType = 'azure';
        _apiKey = azureKey;
        _azureEndpoint = azureEp;
        _azureDeployment = azureDep;
        _initServices();
        return;
    }

    // Check for Gemini Env
    final geminiKey = Platform.environment['GEMINI_API_KEY'] ?? const String.fromEnvironment('GEMINI_API_KEY');
    if (geminiKey.isNotEmpty) {
         print('Auto-configuring Gemini from Environment');
         _providerType = 'gemini';
         _apiKey = geminiKey;
         _initServices();
         return;
    }
  }

  Future<void> _showConfigDialog() async {
    await showDialog(
      context: context,
      barrierDismissible: false,
      builder: (context) {
        return StatefulBuilder(
          builder: (context, setState) {
            return AlertDialog(
              title: const Text('Configure AI Provider'),
              content: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    DropdownButton<String>(
                      value: _providerType,
                      items: const [
                        DropdownMenuItem(value: 'gemini', child: Text('Google Gemini')),
                        DropdownMenuItem(value: 'azure', child: Text('Azure OpenAI')),
                      ],
                      onChanged: (v) => setState(() => _providerType = v!),
                    ),
                    const SizedBox(height: 10),
                    TextField(
                      decoration: const InputDecoration(labelText: 'API Key'),
                      onChanged: (v) => _apiKey = v,
                    ),
                    if (_providerType == 'azure') ...[
                      const SizedBox(height: 10),
                      TextField(
                        decoration: const InputDecoration(labelText: 'Endpoint URL (https://...)'),
                        onChanged: (v) => _azureEndpoint = v,
                      ),
                      const SizedBox(height: 10),
                      TextField(
                        decoration: const InputDecoration(labelText: 'Deployment Name'),
                        onChanged: (v) => _azureDeployment = v,
                      ),
                      const SizedBox(height: 10),
                      TextFormField(
                        initialValue: _azureApiVersion,
                        decoration: const InputDecoration(labelText: 'API Version'),
                        onChanged: (v) => _azureApiVersion = v,
                      ),
                    ]
                  ],
                ),
              ),
              actions: [
                TextButton(
                  onPressed: () {
                    // Validation
                    if (_apiKey == null || _apiKey!.isEmpty) return;

                    if (_providerType == 'azure') {
                      if (_azureEndpoint == null || _azureEndpoint!.isEmpty) return;
                      if (_azureDeployment == null || _azureDeployment!.isEmpty) return;
                    }

                    _initServices();
                    Navigator.pop(context);
                  },
                  child: const Text('Connect'),
                ),
              ],
            );
          },
        );
      },
    );
  }

  Future<void> _initServices() async {
    // Get path from environment
    String? mcpPath = Platform.environment['MCP_SERVER_PATH'];
    if (mcpPath == null || mcpPath.isEmpty) {
      debugPrint('MCP_SERVER_PATH is not set');
      setState(() {
        _messages.add({
          'role': 'system',
          'content': 'Error: MCP_SERVER_PATH environment variable is missing.'
        });
      });
      return;
    }

    print('Starting MCP Client with server at: $mcpPath');

    // Get Directory Server Address
    final dirServerAddr = Platform.environment['DIRECTORY_CLIENT_SERVER_ADDRESS'] ??
                          const String.fromEnvironment('DIRECTORY_CLIENT_SERVER_ADDRESS');

    final oasfSchemaUrl = Platform.environment['OASF_API_VALIDATION_SCHEMA_URL'] ??
                          const String.fromEnvironment('OASF_API_VALIDATION_SCHEMA_URL');

    Map<String, String>? mcpEnv;
    if (dirServerAddr.isNotEmpty) {
      print('Configuring Directory Node at: $dirServerAddr');
      mcpEnv = {'DIRECTORY_CLIENT_SERVER_ADDRESS': dirServerAddr};
    }

    if (oasfSchemaUrl.isNotEmpty) {
       mcpEnv ??= {};
       mcpEnv['OASF_API_VALIDATION_SCHEMA_URL'] = oasfSchemaUrl;
    }

    _mcpClient = McpClient(executablePath: mcpPath);
    try {
      await _mcpClient!.start(environment: mcpEnv);
      await _mcpClient!.initialize();

      _aiService = AiService(mcpClient: _mcpClient!);

      LlmProvider provider;
      if (_providerType == 'azure') {
         provider = AzureOpenAiProvider(
           apiKey: _apiKey!,
           endpoint: _azureEndpoint!,
           deploymentId: _azureDeployment!,
           apiVersion: _azureApiVersion,
         );
      } else {
         provider = GeminiProvider(apiKey: _apiKey!);
      }

      await _aiService!.init(provider);

      setState(() {});
    } catch (e) {
      _addSystemMessage('Error initializing services: $e');
    }
  }

  @override
  void dispose() {
    _mcpClient?.stop();
    super.dispose();
  }

  void _addSystemMessage(String text) {
    setState(() {
      _messages.add({'role': 'system', 'text': text});
    });
  }

  Future<void> _sendMessage() async {
    if (_controller.text.isEmpty || _aiService == null) return;

    final text = _controller.text;
    _controller.clear();

    setState(() {
      _messages.add({'role': 'user', 'text': text});
      _isLoading = true;
    });

    try {
      final responseText = await _aiService!.sendMessage(
        text,
        _history,
        onToolOutput: (name, data) {
          setState(() {
            if (data is List) {
               for (var item in data) {
                 if (item is Map) {
                    _messages.add({
                      'role': 'record',
                      'data': Map<String, dynamic>.from(item),
                      'source': name
                    });
                 }
               }
            } else if (data is Map) {
               _messages.add({
                 'role': 'record',
                 'data': Map<String, dynamic>.from(data),
                 'source': name
               });
            }
          });
        },
      );

      setState(() {
        _history.add(Content.text(text));
        _history.add(Content.model([TextPart(responseText ?? '')]));
        _messages.add({'role': 'model', 'text': responseText});
      });
    } catch (e) {
      setState(() {
        _messages.add({'role': 'error', 'text': e.toString()});
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('GenUI MCP Client'),
        actions: [
          IconButton(
            icon: Icon(Theme.of(context).brightness == Brightness.dark
                ? Icons.light_mode
                : Icons.dark_mode),
            onPressed: () => MyApp.toggleTheme(context),
            tooltip: 'Toggle Theme',
          ),
        ],
      ),
      body: SelectionArea(
        child: Column(
          children: [
            Expanded(
              child: ListView.builder(
                itemCount: _messages.length,
                itemBuilder: (context, index) {
                  final msg = _messages[index];
                  final role = msg['role'];

                  if (role == 'record') {
                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16.0),
                      child: RecordCard(
                        data: msg['data'],
                        title: 'Record from ${msg['source']}',
                      ),
                    );
                  }

                  if (role == 'record_grid') {
                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16.0),
                      child: RecordGrid(
                        items: msg['items'],
                        source: msg['source'],
                      ),
                    );
                  }

                  final text = msg['text'] ?? '';

                  return ListTile(
                    title: Text(role.toUpperCase(), style: const TextStyle(fontWeight: FontWeight.bold)),
                    subtitle: MarkdownBody(
                      data: text,
                      selectable: true,
                    ),
                    tileColor: role == 'user' ? Theme.of(context).colorScheme.surfaceContainerHighest.withOpacity(0.5) : null,
                  );
                },
              ),
            ),
            if (_isLoading) const LinearProgressIndicator(),
            Padding(
              padding: const EdgeInsets.all(8.0),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _controller,
                      decoration: const InputDecoration(
                        hintText: 'Ask something...',
                        border: OutlineInputBorder(),
                      ),
                      onSubmitted: (_) => _sendMessage(),
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.send),
                    onPressed: _sendMessage,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
