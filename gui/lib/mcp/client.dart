import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'model.dart';

typedef ProcessStarter = Future<Process> Function(String executable, List<String> arguments, {Map<String, String>? environment});

class McpClient {
  final String executablePath;
  Process? _process;
  int _idCounter = 0;
  final Map<int, Completer<dynamic>> _pendingRequests = {};
  final ProcessStarter _processStarter;

  McpClient({
    required this.executablePath,
    ProcessStarter? processStarter,
  }) : _processStarter = processStarter ?? Process.start;

  Future<void> start({Map<String, String>? environment}) async {
    final env = {
      "OASF_API_VALIDATION_DISABLE": "true",
    };
    if (environment != null) {
      env.addAll(environment);
    }

    _process = await _processStarter(
      executablePath,
      [],
      environment: env,
    );

    // Handle stdout (responses)
    _process!.stdout
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen((line) {
      if (line.trim().isEmpty) return;
      try {
        final Map<String, dynamic> message = jsonDecode(line);
        _handleMessage(message);
      } catch (e) {
        print('Error decoding MCP message: $e\nLine: $line');
      }
    });

    // Handle stderr
    _process!.stderr.transform(utf8.decoder).listen((data) {
      print('MCP Server Stderr: $data');
    });
  }

  void _handleMessage(Map<String, dynamic> message) {
    if (message.containsKey('id')) {
      final id = message['id'];
      if (_pendingRequests.containsKey(id)) {
        // It's a response to our request
        final completer = _pendingRequests.remove(id)!;
        if (message.containsKey('error')) {
          completer.completeError(message['error']);
        } else {
          completer.complete(message['result']);
        }
      } else {
         // It's a request from the server (e.g. sampling, ping)
         // For now, we don't support server-initiated requests, so we send an error
         // or just ignore if we don't want to complicate things.
         // But per JSON-RPC, we should reply.
         _sendError(id, -32601, "Method not found");
      }
    } else {
      // Notification
      print('MCP Notification: $message');
    }
  }

  void _sendError(dynamic id, int code, String message) {
     if (_process == null) return;
     final response = {
       'jsonrpc': '2.0',
       'id': id,
       'error': {
         'code': code,
         'message': message,
       }
     };
     _process!.stdin.writeln(jsonEncode(response));
  }


  Future<dynamic> _sendRequest(String method, [dynamic params]) async {
    if (_process == null) throw Exception('MCP Client not started');

    final id = _idCounter++;
    final request = {
      'jsonrpc': '2.0',
      'id': id,
      'method': method,
      if (params != null) 'params': params,
    };

    final completer = Completer<dynamic>();
    _pendingRequests[id] = completer;

    _process!.stdin.writeln(jsonEncode(request));

    return completer.future;
  }

  Future<void> initialize() async {
    // Basic initialization based on MCP spec
    await _sendRequest('initialize', {
      'protocolVersion': '0.1.0',
      'capabilities': {},
      'clientInfo': {
        'name': 'dart-genui-client',
        'version': '1.0.0',
      }
    });

    // Send initialized notification
    final initializedMsg = {
      'jsonrpc': '2.0',
      'method': 'notifications/initialized',
    };

    _process!.stdin.writeln(jsonEncode(initializedMsg));
  }

  Future<dynamic> _simulateWebRequest(String method, dynamic params) async {
    // Simulate delay
    await Future.delayed(const Duration(milliseconds: 500));

    if (method == 'initialize') {
      return {
        'protocolVersion': '0.1.0',
        'capabilities': {},
        'serverInfo': {'name': 'dir-mcp-server-web-mock', 'version': '1.0.0'}
      };
    }

    if (method == 'tools/list') {
      return {
        'tools': [
          {
            "name": "agntcy_dir_search_local",
            "description": "Searches for agent records on the local directory node using structured query filters.",
            "inputSchema": {
              "type": "object",
              "properties": {
                "names": {"type": "array", "items": {"type": "string"}},
                "skill_names": {"type": "array", "items": {"type": "string"}},
                "limit": {"type": "integer"}
              }
            }
          },
          {
            "name": "agntcy_dir_pull_record",
            "description": "Pulls an OASF agent record from the local Directory node by its CID.",
            "inputSchema": {
              "type": "object",
              "required": ["cid"],
              "properties": {
                "cid": {"type": "string"}
              }
            }
          }
        ]
      };
    }

    if (method == 'tools/call') {
      final name = params['name'];
      if (name == 'agntcy_dir_search_local') {
        return {
          'content': [
            {
              'type': 'text',
              'text': jsonEncode({
                'count': 1,
                'has_more': false,
                'record_cids': ['QmMockContentIdentifier12345'],
                'message': 'This is a mock response from the Web Client'
              })
            }
          ]
        };
      }
      return {'content': [{'type': 'text', 'text': 'Mock result for $name'}]};
    }

    throw Exception('Unknown method: $method');
  }

  Future<List<McpTool>> listTools() async {
    final result = await _sendRequest('tools/list');
    final tools = (result['tools'] as List).map((t) => McpTool.fromJson(t)).toList();
    return tools;
  }

  Future<McpToolResult> callTool(String name, Map<String, dynamic> arguments) async {
    try {
      final result = await _sendRequest('tools/call', {
        'name': name,
        'arguments': arguments,
      });

      // MCP 0.1.0 draft returns { content: [{type: 'text', text: '...'}] }
      return McpToolResult(content: result['content']);
    } catch (e) {
      return McpToolResult(content: e.toString(), isError: true);
    }
  }

  Future<List<McpPrompt>> listPrompts() async {
    final result = await _sendRequest('prompts/list');
    final prompts = (result['prompts'] as List)
        .map((p) => McpPrompt.fromJson(p))
        .toList();
    return prompts;
  }

  Future<McpGetPromptResult> getPrompt(String name, [Map<String, String>? arguments]) async {
    final result = await _sendRequest('prompts/get', {
      'name': name,
      if (arguments != null) 'arguments': arguments,
    });
    return McpGetPromptResult.fromJson(result);
  }

  Future<void> stop() async {
    _process?.kill();
    _process = null;
  }
}
