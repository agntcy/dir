import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/client.dart';

// Mock Process
class MockProcess implements Process {
  final StreamController<List<int>> stdoutController = StreamController();
  final StreamController<List<int>> stderrController = StreamController();
  final MockIOSink mockStdin = MockIOSink();

  @override
  Stream<List<int>> get stdout => stdoutController.stream;

  @override
  Stream<List<int>> get stderr => stderrController.stream;

  @override
  IOSink get stdin => mockStdin;

  @override
  bool kill([ProcessSignal signal = ProcessSignal.sigterm]) => true;

  @override
  Future<int> get exitCode => Future.value(0);

  // Unused implementations
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// Mock IOSink
class MockIOSink implements IOSink {
  final List<String> writes = [];

  @override
  void writeln([Object? obj = ""]) {
    writes.add(obj.toString());
  }

  @override
  void add(List<int> data) {}

  @override
  void write(Object? obj) {}

  @override
  void writeAll(Iterable objects, [String separator = ""]) {}

  @override
  void writeCharCode(int charCode) {}

  @override
  Future get done => Future.value();

  @override
  Future close() => Future.value();

  @override
  Encoding encoding = utf8;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  late McpClient client;
  late MockProcess mockProcess;

  setUp(() {
    mockProcess = MockProcess();
    client = McpClient(
      executablePath: 'dummy',
      processStarter: (path, args, {environment}) async => mockProcess,
    );
  });

  tearDown(() {
    mockProcess.stdoutController.close();
    mockProcess.stderrController.close();
  });

  group('McpClient Unit Tests', () {
    test('start launches process', () async {
      await client.start();
      expect(mockProcess.mockStdin.writes, isEmpty);
    });

    test('handles JSON-RPC response', () async {
      await client.start();

      // Simulate sending a request
      final future = client.listTools();

      // Verify request was sent to stdin
      expect(mockProcess.mockStdin.writes, isNotEmpty);
      final requestJson = jsonDecode(mockProcess.mockStdin.writes.last);
      final id = requestJson['id'];

      // Simulate response from server
      final response = {
        'jsonrpc': '2.0',
        'id': id,
        'result': {
          'tools': [{'name': 'test_tool', 'description': 'desc', 'inputSchema': {}}]
        }
      };

      mockProcess.stdoutController.add(utf8.encode(jsonEncode(response) + '\n'));

      final tools = await future;
      expect(tools.length, 1);
      expect(tools.first.name, 'test_tool');
    });

    test('handles JSON-RPC error response', () async {
      await client.start();
      final future = client.listTools();

      final requestJson = jsonDecode(mockProcess.mockStdin.writes.last);
      final id = requestJson['id'];

      final response = {
        'jsonrpc': '2.0',
        'id': id,
        'error': {'code': -1, 'message': 'Failed'}
      };

      mockProcess.stdoutController.add(utf8.encode(jsonEncode(response) + '\n'));

      try {
        await future;
        fail('Should have thrown');
      } catch (e) {
        expect(e, isA<Map>());
        expect((e as Map)['code'], -1);
      }
    });

    test('ignores non-JSON lines and logs', () async {
      final logs = <String>[];
      await runZoned(
        () async {
          await client.start();

          // This should be logged but not crash
          mockProcess.stdoutController.add(utf8.encode('[INFO] Server started\n'));

          // This is malformed JSON
          mockProcess.stdoutController.add(utf8.encode('{ "broken": \n'));

          // Allow async processing to happen
          await Future.delayed(Duration.zero);
        },
        zoneSpecification: ZoneSpecification(
          print: (self, parent, zone, line) {
            logs.add(line);
          },
        ),
      );

      expect(logs.any((l) => l.contains('MCP Server Log: [INFO] Server started')), isTrue);
      expect(logs.any((l) => l.contains('Error decoding MCP message')), isTrue);
    });

    test('handles notifications', () async {
      await client.start();

      final notification = {
        'jsonrpc': '2.0',
        'method': 'notifications/test',
        'params': {}
      };

      // Should just print to console (handled in _handleMessage)
      mockProcess.stdoutController.add(utf8.encode(jsonEncode(notification) + '\n'));
    });

    test('handles server-initiated request with error', () async {
      await client.start();

      final request = {
        'jsonrpc': '2.0',
        'id': 999,
        'method': 'server/ping',
        'params': {}
      };

      // Client should respond with "Method not found"
      mockProcess.stdoutController.add(utf8.encode(jsonEncode(request) + '\n'));

      // Wait for async processing
      await Future.delayed(Duration(milliseconds: 10));

      final response = jsonDecode(mockProcess.mockStdin.writes.last);
      expect(response['id'], 999);
      expect(response['error']['code'], -32601);
    });

    test('initialize sends correct handshake', () async {
      await client.start();
      final future = client.initialize();

      // Request 1: initialize
      final req1 = jsonDecode(mockProcess.mockStdin.writes[0]);
      expect(req1['method'], 'initialize');
      expect(req1['params']['protocolVersion'], '0.1.0');

      // Fake response
      mockProcess.stdoutController.add(utf8.encode(jsonEncode({
        'jsonrpc': '2.0',
        'id': req1['id'],
        'result': {'protocolVersion': '0.1.0'}
      }) + '\n'));

      await future;

      // Request 2: initialized notification
      final req2 = jsonDecode(mockProcess.mockStdin.writes[1]);
      expect(req2['method'], 'notifications/initialized');
      expect(req2.containsKey('id'), false);
    });

    test('callTool sends request and returns result', () async {
      await client.start();
      final future = client.callTool('my_tool', {'arg': 1});

      final req = jsonDecode(mockProcess.mockStdin.writes.last);
      expect(req['method'], 'tools/call');
      expect(req['params']['name'], 'my_tool');

      mockProcess.stdoutController.add(utf8.encode(jsonEncode({
        'jsonrpc': '2.0',
        'id': req['id'],
        'result': {'content': [{'type': 'text', 'text': 'ok'}]}
      }) + '\n'));

      final result = await future;
      expect(result.isError, false);
      expect((result.content as List).first['text'], 'ok');
    });

    test('callTool handles exceptions as error result', () async {
      await client.start();
      final future = client.callTool('broken_tool', {});

      final req = jsonDecode(mockProcess.mockStdin.writes.last);
      mockProcess.stdoutController.add(utf8.encode(jsonEncode({
        'jsonrpc': '2.0',
        'id': req['id'],
        'error': {'code': 1, 'message': 'Oops'}
      }) + '\n'));

      final result = await future;
      expect(result.isError, true);
      expect(result.content.toString(), contains('Oops'));
    });

    test('listPrompts parses response', () async {
       await client.start();
      final future = client.listPrompts();

      final req = jsonDecode(mockProcess.mockStdin.writes.last);
      expect(req['method'], 'prompts/list');

      mockProcess.stdoutController.add(utf8.encode(jsonEncode({
        'jsonrpc': '2.0',
        'id': req['id'],
        'result': {
          'prompts': [{'name': 'p1', 'description': 'd', 'arguments': []}]
        }
      }) + '\n'));

      final prompts = await future;
      expect(prompts.first.name, 'p1');
    });

    test('getPrompt sends arguments', () async {
       await client.start();
      final future = client.getPrompt('p1', {'k': 'v'});

      final req = jsonDecode(mockProcess.mockStdin.writes.last);
      expect(req['method'], 'prompts/get');
      expect(req['params']['arguments']['k'], 'v');

      mockProcess.stdoutController.add(utf8.encode(jsonEncode({
        'jsonrpc': '2.0',
        'id': req['id'],
        'result': {
          'description': 'd',
          'messages': []
        }
      }) + '\n'));

      final result = await future;
      expect(result.description, 'd');
    });

    test('stop kills process', () async {
      await client.start();
      // We can't easily verify the mock's internals via the interface,
      // but we can ensure no crash
      await client.stop();
    });
  });
}
