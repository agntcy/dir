// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../main.dart';
import '../mcp/client.dart';
import '../services/ai_service.dart';
import '../services/analytics_service.dart';
import '../services/llm_provider.dart';
import 'settings_screen.dart';
import 'widgets/record_card.dart';
import 'widgets/search_results_widget.dart';

class ChatScreen extends StatefulWidget {
  final AiService? aiService;

  const ChatScreen({super.key, this.aiService});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _controller = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final List<Content> _history = [];
  final List<Map<String, dynamic>> _messages = []; // For UI display

  // Track pulled records by CID for associating with search results
  final Map<String, Map<String, dynamic>> _pulledRecords = {};
  String? _pendingPullCid; // CID currently being pulled

  AiService? _aiService;
  McpClient? _mcpClient;
  bool _isLoading = false;

  // Welcome message
  static const String _welcomeMessage = 'Find published AI agents records by describing what you need (by author, name, skills, versions, domain). Or type `help` for commands.';

  // Config
  bool _isConfigured = false;
  String _providerType = 'gemini'; // gemini, azure, openai
  String? _apiKey;
  String? _azureEndpoint;
  String? _azureDeployment;
  String _azureApiVersion = '2024-10-21'; // Default
  String? _openaiEndpoint; // For OpenAI-compatible gateways

  // Directory Config
  String? _directoryUrl;
  String? _directoryToken;
  String? _oasfSchemaUrl;

  String? _ollamaEndpoint;
  String? _ollamaModel;

  @override
  void initState() {
    super.initState();
    // Add welcome message
    _messages.add({'role': 'welcome', 'text': _welcomeMessage});

    if (widget.aiService != null) {
      _aiService = widget.aiService;
      _isConfigured = true;
    } else {
      _initConfig();
    }
  }

  Future<void> _initConfig() async {
    final prefs = await SharedPreferences.getInstance();
    final storedProvider = prefs.getString('provider');

    if (storedProvider != null && storedProvider.isNotEmpty) {
      // Load from Prefs
      setState(() {
        _providerType = storedProvider;
        if (_providerType == 'gemini') {
          _apiKey = prefs.getString('gemini_api_key');
        } else if (_providerType == 'azure') {
          _apiKey = prefs.getString('azure_api_key');
          _azureEndpoint = prefs.getString('azure_endpoint');
          _azureDeployment = prefs.getString('azure_deployment');
          _azureApiVersion = prefs.getString('azure_api_version') ?? '2024-10-21';
        } else if (_providerType == 'openai') {
          _apiKey = prefs.getString('openai_api_key');
          _openaiEndpoint = prefs.getString('openai_endpoint');
        } else if (_providerType == 'ollama') {
          _ollamaEndpoint = prefs.getString('ollama_endpoint');
          _ollamaModel = prefs.getString('ollama_model');
        }

        // Load Directory Config
        _directoryUrl = prefs.getString('directory_server_address');
        _directoryToken = prefs.getString('directory_github_token');
        _oasfSchemaUrl = prefs.getString('oasf_schema_url');
      });

      if ((_providerType == 'ollama') || (_apiKey != null && _apiKey!.isNotEmpty)) {
        print('Configured $_providerType from Settings');
        setState(() => _isConfigured = true);
        _initServices();
        return;
      }
    }

    // Fallback to Env
    _checkEnvAndInit();
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    _mcpClient?.stop();
    super.dispose();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 800),
          curve: Curves.easeInOutQuart,
        );
      }
    });
  }

  void _checkEnvAndInit() {
    // Check for OpenAI-compatible gateway (e.g., Outshift AI Gateway)
    final openaiKey = Platform.environment['OPENAI_API_KEY'] ?? const String.fromEnvironment('OPENAI_API_KEY');
    final openaiEndpoint = Platform.environment['OPENAI_ENDPOINT'] ?? const String.fromEnvironment('OPENAI_ENDPOINT');

    if (openaiKey.isNotEmpty && openaiEndpoint.isNotEmpty) {
        print('Auto-configuring OpenAI-compatible gateway from Environment');
        _providerType = 'openai';
        _apiKey = openaiKey;
        _openaiEndpoint = openaiEndpoint;
        setState(() => _isConfigured = true);
        _initServices();
        return;
    }

    // Check for Azure Env (Runtime env preferred, then compile-time)
    final azureKey = Platform.environment['AZURE_API_KEY'] ?? const String.fromEnvironment('AZURE_API_KEY');
    final azureEp = Platform.environment['AZURE_ENDPOINT'] ?? const String.fromEnvironment('AZURE_ENDPOINT');
    final azureDep = Platform.environment['AZURE_DEPLOYMENT'] ?? const String.fromEnvironment('AZURE_DEPLOYMENT');
    final azureApiVer = Platform.environment['AZURE_OPENAI_API_VERSION'] ?? const String.fromEnvironment('AZURE_OPENAI_API_VERSION');

    if (azureKey.isNotEmpty && azureEp.isNotEmpty && azureDep.isNotEmpty) {
        print('Auto-configuring Azure from Environment');
        _providerType = 'azure';
        _apiKey = azureKey;
        _azureEndpoint = azureEp;
        _azureDeployment = azureDep;
        if (azureApiVer.isNotEmpty) {
           _azureApiVersion = azureApiVer;
        }
        setState(() => _isConfigured = true);
        _initServices();
        return;
    }

    // Check for Gemini Env
    final geminiKey = Platform.environment['GEMINI_API_KEY'] ?? const String.fromEnvironment('GEMINI_API_KEY');
    if (geminiKey.isNotEmpty) {
         print('Auto-configuring Gemini from Environment');
         _providerType = 'gemini';
         _apiKey = geminiKey;
         setState(() => _isConfigured = true);
         _initServices();
         return;
    }

    // Check for Ollama (local) - implicit if configured?
    // Usually local usage doesn't need env vars unless we want to force it.
    // We'll skip auto-init for Ollama unless specific ENV is present to force it.
    final ollamaEp = Platform.environment['OLLAMA_ENDPOINT'] ?? const String.fromEnvironment('OLLAMA_ENDPOINT');
    if (ollamaEp.isNotEmpty) {
        print('Auto-configuring Ollama from Environment');
        _providerType = 'ollama';
        _ollamaEndpoint = ollamaEp;
        _ollamaModel = Platform.environment['OLLAMA_MODEL'] ?? const String.fromEnvironment('OLLAMA_MODEL');
        if (_ollamaModel == null || _ollamaModel!.isEmpty) _ollamaModel = 'gemma3:4b'; // Default

        setState(() => _isConfigured = true);
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
                        DropdownMenuItem(value: 'ollama', child: Text('Ollama')),
                      ],
                      onChanged: (v) => setState(() => _providerType = v!),
                    ),
                    const SizedBox(height: 10),
                    if (_providerType != 'ollama')
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
                    ],
                    if (_providerType == 'ollama') ...[
                      const SizedBox(height: 10),
                      TextField(
                        decoration: const InputDecoration(labelText: 'Endpoint (default: http://localhost:11434/api/chat)'),
                        onChanged: (v) => _ollamaEndpoint = v,
                      ),
                      const SizedBox(height: 10),
                      TextField(
                        decoration: const InputDecoration(labelText: 'Model (default: gemma3:4b)'),
                        onChanged: (v) => _ollamaModel = v,
                      ),
                    ]
                  ],
                ),
              ),
              actions: [
                TextButton(
                  onPressed: () {
                    // Validation
                    if (_providerType != 'ollama' && (_apiKey == null || _apiKey!.isEmpty)) return;

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
    // Get path from environment or search in bundled/dev locations
    String? mcpPath = Platform.environment['MCP_SERVER_PATH'];

    // Debug Mode Details
    debugPrint('Searching for MCP Server...');
    debugPrint('Current Directory: ${Directory.current.path}');
    debugPrint('Resolved Executable: ${Platform.resolvedExecutable}');

    if (mcpPath == null || mcpPath.isEmpty) {
      // 1. Check bundled resource (macOS mostly)
      // Platform.resolvedExecutable points to .../Contents/MacOS/AGNTCY Directory
      final exeDir = File(Platform.resolvedExecutable).parent;
      final macResourcePath = '${exeDir.parent.path}/Resources/mcp-server';
      debugPrint('Checking Bundle Path: $macResourcePath');

      // Determines binary name based on platform
      String binaryName = 'mcp-server';
      if (Platform.isWindows) {
        binaryName = 'mcp-server.exe';
      }

      // 2. Check same directory (Linux/Windows bundled)
      final localPath = '${exeDir.path}/$binaryName';
      debugPrint('Checking Local Path: $localPath');

      // 3. Check development path (relative to gui root)
      // When running 'flutter run', CWD might be the gui root, or nested.
      final devPath = '${Directory.current.path}/../bin/$binaryName';
      debugPrint('Checking Dev Path: $devPath');

      if (await File(macResourcePath).exists()) {
        mcpPath = macResourcePath;
      } else if (await File(localPath).exists()) {
        mcpPath = localPath;
      } else if (await File(devPath).exists()) {
        // Normalize path for clean printing
        mcpPath = File(devPath).absolute.path;
      }
    }

    if (mcpPath == null || mcpPath.isEmpty) {
      debugPrint('MCP_SERVER_PATH is not set and binary not found in default locations');
      if (mounted) {
        setState(() {
          String binaryName = Platform.isWindows ? 'mcp-server.exe' : 'mcp-server';
          _messages.add({
            'role': 'system',
            'text': 'Error: MCP_SERVER_PATH not set and $binaryName binary not found.\nChecked:\n- Bundle Resources\n- App Directory\n- ../bin/$binaryName'
          });
        });
      }
      return;
    }

    print('Starting MCP Client with server at: $mcpPath');

    // Get Directory Server Address - Prefer Settings, then Env
    String dirServerAddr = _directoryUrl ?? '';
    if (dirServerAddr.isEmpty) {
        dirServerAddr = Platform.environment['DIRECTORY_CLIENT_SERVER_ADDRESS'] ??
                          const String.fromEnvironment('DIRECTORY_CLIENT_SERVER_ADDRESS');
    }

    String oasfSchema = _oasfSchemaUrl ?? '';
    if (oasfSchema.isEmpty) {
        oasfSchema = Platform.environment['OASF_API_VALIDATION_SCHEMA_URL'] ??
                          const String.fromEnvironment('OASF_API_VALIDATION_SCHEMA_URL');
    }

    Map<String, String> mcpEnv = {};
    if (dirServerAddr.isNotEmpty) {
      print('Configuring Directory Node at: $dirServerAddr');
      mcpEnv['DIRECTORY_CLIENT_SERVER_ADDRESS'] = dirServerAddr;
    }

    // Auth Token configuration
    if (_directoryToken != null && _directoryToken!.isNotEmpty) {
       mcpEnv['DIRECTORY_CLIENT_GITHUB_TOKEN'] = _directoryToken!;
       // If a token is provided, we assume GitHub auth mode for the token unless specified otherwise
       // In future we might want a dropdown for Auth Mode in settings
       mcpEnv['DIRECTORY_CLIENT_AUTH_MODE'] = 'github';
       print('Configuring Directory Auth Token (using github mode)');
    }

    // OASF Schema URL is required for validation
    if (oasfSchema.isEmpty) {
      if (mounted) {
        setState(() {
          _messages.add({
            'role': 'system',
            'text': 'Error: OASF Schema URL is required for validation. Please configure it in Settings.',
          });
        });
      }
      return;
    }
    mcpEnv['OASF_API_VALIDATION_SCHEMA_URL'] = oasfSchema;

    _mcpClient = McpClient(executablePath: mcpPath);
    try {
      print('DEBUG: Starting MCP client...');
      await _mcpClient!.start(environment: mcpEnv);
      print('DEBUG: MCP client started, initializing...');
      await _mcpClient!.initialize();
      print('DEBUG: MCP client initialized');

      _aiService = AiService(mcpClient: _mcpClient!);
      print('DEBUG: AiService created');

      LlmProvider provider;
      if (_providerType == 'openai') {
         print('DEBUG: Creating OpenAI-compatible provider with endpoint: $_openaiEndpoint');
         provider = OpenAiCompatibleProvider(
           apiKey: _apiKey!,
           endpoint: _openaiEndpoint!,
         );
      } else if (_providerType == 'azure') {
         provider = AzureOpenAiProvider(
           apiKey: _apiKey!,
           endpoint: _azureEndpoint!,
           deploymentId: _azureDeployment!,
           apiVersion: _azureApiVersion,
         );
      } else if (_providerType == 'ollama') {
         provider = OllamaProvider(
            endpoint: _ollamaEndpoint?.isEmpty ?? true ? 'http://localhost:11434/api/chat' : _ollamaEndpoint!,
            model: _ollamaModel?.isEmpty ?? true ? 'gemma3:4b' : _ollamaModel!,
         );
      } else {
         provider = GeminiProvider(apiKey: _apiKey!);
      }

      print('DEBUG: Initializing AI service with provider...');
      await _aiService!.init(provider);
      print('DEBUG: AI service initialized successfully!');

      setState(() {
        _messages.add({'role': 'system', 'text': 'Ready! You can now send messages.'});
      });
    } catch (e, stackTrace) {
      print('DEBUG: Error during initialization: $e');
      print('DEBUG: Stack trace: $stackTrace');
      _addSystemMessage('Error initializing services: $e');
    }
  }

  void _addSystemMessage(String text) {
    setState(() {
      _messages.add({'role': 'system', 'text': text});
    });
  }

  /// Extract search criteria from user message for display
  Map<String, dynamic> _extractSearchCriteria(String userMessage) {
    // Simple heuristic extraction - show the user's query
    final criteria = <String, dynamic>{};

    final lowerMsg = userMessage.toLowerCase();

    // Detect keywords
    if (lowerMsg.contains('name')) {
      criteria['filter'] = 'name';
    }
    if (lowerMsg.contains('skill')) {
      criteria['filter'] = 'skill';
    }
    if (lowerMsg.contains('*')) {
      criteria['pattern'] = 'wildcard (*)';
    }
    if (lowerMsg.contains('all')) {
      criteria['scope'] = 'all agents';
    }

    // Always show the query
    criteria['query'] = userMessage.length > 60
        ? '${userMessage.substring(0, 60)}...'
        : userMessage;

    return criteria;
  }

  /// Auto-pull all records to populate agent info in search results
  Future<void> _autoPullRecords(List<String> cids) async {
    if (_aiService == null) return;

    print('DEBUG: Auto-pulling ${cids.length} records...');

    for (final cid in cids) {
      // Skip if already pulled
      if (_pulledRecords.containsKey(cid)) {
        print('DEBUG: Skipping $cid - already pulled');
        continue;
      }

      try {
        print('DEBUG: Pulling record $cid...');
        // Call the MCP tool directly to pull the record
        final result = await _aiService!.mcpClient.callTool(
          'agntcy_dir_pull_record',
          {'cid': cid},
        );

        if (result.isError) {
          print('ERROR: Pull failed for $cid: ${result.content}');
          continue;
        }

        print('DEBUG: Pull result type: ${result.content.runtimeType}');

        // Parse the result - MCP returns [{type: 'text', text: 'json...'}]
        dynamic parsed;
        final content = result.content;

        if (content is List && content.isNotEmpty) {
          for (final item in content) {
            if (item is Map && item['type'] == 'text' && item['text'] != null) {
              try {
                parsed = jsonDecode(item['text'] as String);
                print('DEBUG: Parsed from MCP text content');
                break;
              } catch (e) {
                print('DEBUG: Failed to parse text content: $e');
              }
            }
          }
        } else if (content is String) {
          try {
            parsed = jsonDecode(content);
          } catch (e) {
            print('DEBUG: Failed to parse string content: $e');
          }
        } else if (content is Map) {
          parsed = content;
        }

        if (parsed == null) {
          print('DEBUG: Could not parse content for $cid');
          continue;
        }

        print('DEBUG: Parsed data keys: ${parsed is Map ? (parsed as Map).keys.toList() : 'not a map'}');

        if (parsed is Map) {
          final recordData = Map<String, dynamic>.from(parsed);

          // Check for nested data fields - handle both 'data' and 'record_data' keys
          Map<String, dynamic> agentData;
          if (recordData.containsKey('record_data')) {
            // record_data might be a JSON string or a Map
            final recordDataValue = recordData['record_data'];
            if (recordDataValue is String) {
              try {
                agentData = Map<String, dynamic>.from(jsonDecode(recordDataValue));
                print('DEBUG: Parsed record_data from JSON string');
              } catch (e) {
                print('DEBUG: Failed to parse record_data string: $e');
                agentData = recordData;
              }
            } else if (recordDataValue is Map) {
              agentData = Map<String, dynamic>.from(recordDataValue);
              print('DEBUG: Found record_data as Map');
            } else {
              agentData = recordData;
            }
          } else if (recordData.containsKey('data') && recordData['data'] is Map) {
            print('DEBUG: Found nested data, extracting agent info');
            agentData = Map<String, dynamic>.from(recordData['data'] as Map);
          } else {
            agentData = recordData;
          }

          agentData['cid'] = cid;
          _pulledRecords[cid] = agentData;

          print('DEBUG: Stored record for $cid with name: ${agentData['name']}');

          // Update UI to show loaded data
          if (mounted) {
            setState(() {});
          }
        }
      } catch (e, stackTrace) {
        print('ERROR: Auto-pulling record $cid: $e');
        print('Stack: $stackTrace');
      }
    }
    print('DEBUG: Auto-pull complete. Pulled records count: ${_pulledRecords.length}');
  }

  Future<void> _sendMessage() async {
    if (_controller.text.isEmpty || _aiService == null) return;

    final text = _controller.text;
    _controller.clear();

    setState(() {
      _messages.add({'role': 'user', 'text': text});
      _isLoading = true;
    });
    _scrollToBottom();

    // Log message event
    AnalyticsService().logEvent('send_message', params: {
      'provider': _providerType,
      'model': _providerType == 'ollama' ? _ollamaModel : (_providerType == 'azure' ? _azureDeployment : 'gemini'),
      'has_context': _history.isNotEmpty,
    });

    try {
      final responseText = await _aiService!.sendMessage(
        text,
        _history,
        onToolOutput: (name, data) {
          AnalyticsService().logEvent('tool_use', params: {'tool_name': name});
          setState(() {
            if (data is Map) {
              final mapData = Map<String, dynamic>.from(data);

              // Check if this is a search result
              if (name == 'agntcy_dir_search_local' && mapData.containsKey('count')) {
                // Remove any previous search_results from this conversation turn
                // to avoid duplicate search widgets
                _messages.removeWhere((m) => m['role'] == 'search_results');

                _messages.add({
                  'role': 'search_results',
                  'data': mapData,
                  'source': name,
                  'searchCriteria': _extractSearchCriteria(text),
                });

                // Auto-pull all records to get full agent data
                // Use Future.microtask to schedule this outside of setState
                final cids = (mapData['record_cids'] as List?)?.cast<String>() ?? [];
                Future.microtask(() => _autoPullRecords(cids));
              }
              // Check if this is a pulled record
              else if (name == 'agntcy_dir_pull_record' && mapData.containsKey('data')) {
                // Try to get CID from pending, or extract from user message
                String cid = _pendingPullCid ?? '';

                // If no pending CID, try to extract from the last user message
                if (cid.isEmpty) {
                  for (int i = _messages.length - 1; i >= 0; i--) {
                    final msg = _messages[i];
                    if (msg['role'] == 'user' && msg['text'] != null) {
                      final userText = msg['text'].toString();
                      // Extract CID from "Pull the record with CID: X" pattern
                      final cidMatch = RegExp(r'CID[:\s]+([a-z0-9]+)', caseSensitive: false)
                          .firstMatch(userText);
                      if (cidMatch != null) {
                        cid = cidMatch.group(1) ?? '';
                      }
                      break;
                    }
                  }
                }

                final recordWithCid = {'cid': cid, ...mapData};

                // Store in pulled records map
                if (cid.isNotEmpty) {
                  _pulledRecords[cid] = recordWithCid;
                  _pendingPullCid = null;
                }

                // Show expandable detail card (expanded by default)
                _messages.add({
                  'role': 'agent_record',
                  'data': recordWithCid,
                  'source': name,
                });
              }
              // Generic record
              else {
                _messages.add({
                  'role': 'record',
                  'data': mapData,
                  'source': name
                });
              }
            } else if (data is List) {
               for (var item in data) {
                 if (item is Map) {
                    _messages.add({
                      'role': 'record',
                      'data': Map<String, dynamic>.from(item),
                      'source': name
                    });
                 }
               }
             }
          });
          // Don't scroll during tool processing - only after final response
        },
      );

      setState(() {
        _history.add(Content.text(text));
        _history.add(Content.model([TextPart(responseText ?? '')]));
        _messages.add({'role': 'model', 'text': responseText});
      });
      _scrollToBottom();
    } catch (e) {
      setState(() {
        _messages.add({'role': 'error', 'text': e.toString()});
      });
      _scrollToBottom();
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  Widget _buildSuggestionChip(BuildContext context, String text) {
    return ActionChip(
      label: Text(
        text,
        style: TextStyle(
          fontSize: 12,
          color: Theme.of(context).colorScheme.primary,
        ),
      ),
      backgroundColor: Theme.of(context).colorScheme.primary.withOpacity(0.1),
      side: BorderSide(
        color: Theme.of(context).colorScheme.primary.withOpacity(0.3),
      ),
      onPressed: () {
        if (text.toLowerCase() == 'help') {
          _showHelpCommands();
        } else {
          _controller.text = text;
          _sendMessage();
        }
      },
    );
  }

  void _showHelpCommands() {
    setState(() {
      _messages.add({
        'role': 'user',
        'text': 'help',
      });
      _messages.add({
        'role': 'model',
        'text': '''## Commands

**Search Agents**

- List of all agents - Show all registered agents
- search for \<query\> - Search agents by name, author, or description
- find agents with skill \<skill\> - Search by skill
- agents by author \<name\> - Filter by author

**Agent Details**

- Click "See more" on any agent card to view full details
- Click the CID badge to copy the agent identifier
- Click "Download JSON" to save agent data

**Examples**

- "agents for text summarization"
- "search by author cisco"
- "list agents with problem solving skills"

Type your query or click a suggestion to get started!''',
      });
    });
  }

  Future<void> _openSettings() async {
    final result = await Navigator.push<bool>(
      context,
      MaterialPageRoute(builder: (context) => const SettingsScreen()),
    );

    if (result == true) {
      // Config changed
      setState(() {
        _messages.add({
          'role': 'system',
          'text': 'Configuration updated. Re-connecting services...',
        });
      });

      _aiService = null;
      _mcpClient?.stop(); // Ensure old client is stopped
      _mcpClient = null;

      await _initConfig();
    }
  }

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: AppBar(
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Blue logo for light mode, orange logo for dark mode
            SvgPicture.asset(
              isDark
                  ? 'assets/images/logo-dark.svg'   // Orange on dark
                  : 'assets/images/logo-light.svg', // Blue on light
              height: 28,
            ),
            const SizedBox(width: 12),
            const Text(
              'Agent Directory GUI',
              style: TextStyle(
                fontWeight: FontWeight.w600,
                fontSize: 16,
                letterSpacing: -0.3,
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.settings),
            tooltip: 'Settings',
            onPressed: _openSettings,
          ),
          IconButton(
            icon: Icon(isDark ? Icons.light_mode : Icons.dark_mode),
            onPressed: () => MyApp.toggleTheme(context),
            tooltip: 'Toggle Theme',
          ),
        ],
      ),
      body: SelectionArea(
        child: Column(
          children: [
            if (!_isConfigured)
              Container(
                width: double.infinity,
                color: Theme.of(context).colorScheme.errorContainer,
                padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 16),
                child: Row(
                  children: [
                    Icon(Icons.warning_amber_rounded, color: Theme.of(context).colorScheme.error),
                    const SizedBox(width: 8),
                    Expanded(
                       child: Text(
                         'AI Provider not configured. Please check settings.',
                         style: TextStyle(color: Theme.of(context).colorScheme.onErrorContainer, fontWeight: FontWeight.bold),
                       ),
                    ),
                    TextButton(
                      onPressed: _openSettings,
                      child: const Text('Settings'),
                    ),
                  ],
                ),
              ),
            Expanded(
              child: ListView.builder(
                controller: _scrollController,
                physics: const ClampingScrollPhysics(), // No elastic bounce effect
                itemCount: _messages.length,
                itemBuilder: (context, index) {
                  final msg = _messages[index];
                  final role = msg['role'];

                  // Welcome message
                  if (role == 'welcome') {
                    return Container(
                      margin: const EdgeInsets.all(24),
                      padding: const EdgeInsets.all(24),
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: [
                            Theme.of(context).colorScheme.primary.withOpacity(0.05),
                            Theme.of(context).colorScheme.secondary.withOpacity(0.03),
                          ],
                        ),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                          color: Theme.of(context).colorScheme.primary.withOpacity(0.15),
                        ),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Container(
                                padding: const EdgeInsets.all(10),
                                decoration: BoxDecoration(
                                  color: Theme.of(context).colorScheme.primary.withOpacity(0.15),
                                  borderRadius: BorderRadius.circular(12),
                                ),
                                child: Icon(
                                  Icons.search,
                                  color: Theme.of(context).colorScheme.primary,
                                  size: 24,
                                ),
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      'Welcome to AGNTCY Agent Directory GUI',
                                      style: TextStyle(
                                        fontSize: 18,
                                        fontWeight: FontWeight.w700,
                                        color: Theme.of(context).colorScheme.primary,
                                      ),
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      'Discover AI agents from the network',
                                      style: TextStyle(
                                        fontSize: 13,
                                        color: Theme.of(context).colorScheme.onSurface.withOpacity(0.6),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 20),
                          Text(
                            msg['text'] ?? '',
                            style: TextStyle(
                              fontSize: 15,
                              height: 1.6,
                              color: Theme.of(context).colorScheme.onSurface.withOpacity(0.85),
                            ),
                          ),
                          const SizedBox(height: 16),
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: [
                              _buildSuggestionChip(context, 'List of all agents'),
                              _buildSuggestionChip(context, 'Help'),
                            ],
                          ),
                        ],
                      ),
                    );
                  }

                  // New search results widget
                  if (role == 'search_results') {
                    final data = msg['data'] as Map<String, dynamic>;
                    final cids = (data['record_cids'] as List?)?.cast<String>() ?? [];

                    // Collect loaded records from the _pulledRecords map
                    final loadedRecords = <Map<String, dynamic>>[];
                    for (final cid in cids) {
                      if (_pulledRecords.containsKey(cid)) {
                        loadedRecords.add(_pulledRecords[cid]!);
                      }
                    }

                    // Check if still loading (CIDs exist but not all records loaded)
                    final isLoading = cids.isNotEmpty && loadedRecords.length < cids.length;

                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16.0),
                      child: SearchResultsWidget(
                        totalCount: data['count'] ?? 0,
                        hasMore: data['has_more'] ?? false,
                        recordCids: cids,
                        agentRecords: loadedRecords.isNotEmpty ? loadedRecords : null,
                        searchCriteria: msg['searchCriteria'] as Map<String, dynamic>?,
                        errorMessage: data['error_message']?.toString(),
                        isLoading: isLoading,
                        onPullRecord: (cid) {
                          // Track which CID we're pulling
                          _pendingPullCid = cid;
                          // Ask AI to pull the record
                          _controller.text = 'Pull the record with CID: $cid';
                          _sendMessage();
                        },
                      ),
                    );
                  }

                  // Agent record (pulled) - full detail card
                  if (role == 'agent_record') {
                    final data = msg['data'] as Map<String, dynamic>;
                    final cid = data['cid']?.toString() ?? '';

                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16.0),
                      child: AgentDetailCard(
                        cid: cid,
                        agentData: data,
                      ),
                    );
                  }

                  // Record data as JSON code block
                  if (role == 'record') {
                    return Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16.0),
                      child: JsonCodeBlock(
                        data: msg['data'] as Map<String, dynamic>,
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

                  // User message - aligned right, 80% width
                  if (role == 'user') {
                    return Align(
                      alignment: Alignment.centerRight,
                      child: FractionallySizedBox(
                        widthFactor: 0.8,
                        child: Container(
                          margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: Theme.of(context).colorScheme.primaryContainer.withOpacity(0.12),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(
                              color: Theme.of(context).colorScheme.primary.withOpacity(0.2),
                            ),
                          ),
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              CircleAvatar(
                                radius: 14,
                                backgroundColor: Theme.of(context).colorScheme.primary,
                                child: Icon(Icons.person, size: 16, color: Theme.of(context).colorScheme.onPrimary),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('You', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, color: Theme.of(context).colorScheme.primary)),
                                    const SizedBox(height: 4),
                                    Text(text, style: const TextStyle(fontSize: 14)),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    );
                  }

                  // Model response - aligned left, 80% width
                  if (role == 'model') {
                    // Check if response contains JSON data (code block or raw)
                    final jsonBlockMatch = RegExp(r'```(?:json)?\s*([\s\S]*?)\s*```').firstMatch(text);
                    final trimmedText = text.trim();
                    final isRawJson = trimmedText.startsWith('{') && trimmedText.endsWith('}');

                    // Try to extract JSON
                    String textBeforeJson = '';
                    Map<String, dynamic>? jsonData;

                    if (jsonBlockMatch != null) {
                      textBeforeJson = text.substring(0, jsonBlockMatch.start).trim();
                      try {
                        jsonData = jsonDecode(jsonBlockMatch.group(1)!);
                      } catch (_) {}
                    } else if (isRawJson) {
                      try {
                        jsonData = jsonDecode(trimmedText);
                      } catch (_) {}
                    }

                    return Align(
                      alignment: Alignment.centerLeft,
                      child: FractionallySizedBox(
                        widthFactor: 0.8,
                        child: Container(
                          margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: Theme.of(context).colorScheme.surfaceContainerHighest.withOpacity(0.12),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(
                              color: Theme.of(context).colorScheme.outline.withOpacity(0.15),
                            ),
                          ),
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              CircleAvatar(
                                radius: 14,
                                backgroundColor: Theme.of(context).colorScheme.secondary,
                                child: Icon(Icons.smart_toy_outlined, size: 16, color: Theme.of(context).colorScheme.onSecondary),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('Assistant', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, color: Theme.of(context).colorScheme.secondary)),
                                    const SizedBox(height: 4),
                                    // Show text before JSON if any
                                    if (textBeforeJson.isNotEmpty) ...[
                                      MarkdownBody(
                                        data: textBeforeJson,
                                        selectable: true,
                                        styleSheet: MarkdownStyleSheet(
                                          h2: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Theme.of(context).colorScheme.onSurface),
                                          p: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                                          strong: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Theme.of(context).colorScheme.onSurface),
                                          listBullet: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                                        ),
                                      ),
                                      const SizedBox(height: 8),
                                    ],
                                    // Show JSON as code block if detected
                                    if (jsonData != null)
                                      JsonCodeBlock(data: jsonData, title: 'Record Data', maxHeight: 250)
                                    else if (jsonBlockMatch == null)
                                      MarkdownBody(
                                        data: text,
                                        selectable: true,
                                        styleSheet: MarkdownStyleSheet(
                                          h2: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Theme.of(context).colorScheme.onSurface),
                                          p: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                                          strong: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Theme.of(context).colorScheme.onSurface),
                                          listBullet: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                                        ),
                                      ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    );
                  }

                  // System/error messages
                  return ListTile(
                    leading: Icon(
                      role == 'error' ? Icons.error : Icons.info,
                      color: role == 'error' ? Colors.red : Theme.of(context).colorScheme.secondary,
                      size: 20,
                    ),
                    title: Text(
                      role == 'error' ? 'Error' : 'System',
                      style: TextStyle(
                        fontWeight: FontWeight.bold,
                        fontSize: 12,
                        color: role == 'error' ? Colors.red : Theme.of(context).colorScheme.secondary,
                      ),
                    ),
                    subtitle: MarkdownBody(
                      data: text,
                      selectable: true,
                      styleSheet: MarkdownStyleSheet(
                        h2: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Theme.of(context).colorScheme.onSurface),
                        p: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                        strong: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Theme.of(context).colorScheme.onSurface),
                        listBullet: TextStyle(fontSize: 14, color: Theme.of(context).colorScheme.onSurface),
                      ),
                    ),
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
