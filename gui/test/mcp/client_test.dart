// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';

// Mocks
class MockProcess implements Process {
  final StreamController<List<int>> _stdoutController = StreamController();
  final StreamController<List<int>> _stderrController = StreamController();
  final MockIOSink _stdinSink = MockIOSink();

  @override
  Stream<List<int>> get stdout => _stdoutController.stream;

  @override
  Stream<List<int>> get stderr => _stderrController.stream;

  @override
  IOSink get stdin => _stdinSink;

  @override
  Future<int> get exitCode => Future.value(0);

  // Test helpers
  void emitStdout(String data) {
    _stdoutController.add(utf8.encode(data));
  }

  List<String> get writtenLines => _stdinSink.lines;

  @override
  bool kill([ProcessSignal signal = ProcessSignal.sigterm]) => true;

  @override
  int get pid => 1;
}

class MockIOSink implements IOSink {
  final List<String> lines = [];

  @override
  void writeln([Object? obj = ""]) {
    lines.add(obj.toString());
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
  Future addStream(Stream<List<int>> stream) async {}
  @override
  Future close() async {}
  @override
  Future get done => Future.value();
  @override
  Encoding encoding = utf8;
  @override
  void addError(Object error, [StackTrace? stackTrace]) {}
  @override
  Future flush() async {}
}

void main() {
  group('McpClient', () {
    late McpClient client;
    late MockProcess mockProcess;

    setUp(() {
      mockProcess = MockProcess();
      client = McpClient(
        executablePath: 'dummy',
        processStarter: (exe, args, {environment}) async => mockProcess,
      );
    });

    test('initialize sends correct request', () async {
      await client.start();

      // We expect initialize to send a request.
      // Since it awaits a response, we need to simulate the response concurrently
      // or arrange the flow such that we don't dead lock.
      // But _sendRequest awaits. So we need to ensure we can reply.

      final initFuture = client.initialize();

      // Wait a tick for the request to be written
      await Future.delayed(Duration.zero);

      // Check that request was written
      expect(mockProcess.writtenLines.length, 1);
      final request = jsonDecode(mockProcess.writtenLines.first);
      expect(request['method'], 'initialize');
      final id = request['id'];

      // Simulate response
      mockProcess.emitStdout(jsonEncode({
        'jsonrpc': '2.0',
        'id': id,
        'result': {
          'protocolVersion': '0.1.0',
          'capabilities': {},
          'serverInfo': {'name': 'test', 'version': '1.0'}
        }
      }) + '\n');

      await initFuture;

      // Expect initialized notification
      expect(mockProcess.writtenLines.length, 2);
      final notification = jsonDecode(mockProcess.writtenLines.last);
      expect(notification['method'], 'notifications/initialized');
    });

    test('listTools parses tools correctly', () async {
      await client.start();

      final future = client.listTools();
      await Future.delayed(Duration.zero);

      final request = jsonDecode(mockProcess.writtenLines.first);
      expect(request['method'], 'tools/list');

      mockProcess.emitStdout(jsonEncode({
        'jsonrpc': '2.0',
        'id': request['id'],
        'result': {
          'tools': [
            {
              'name': 'test_tool',
              'description': 'A test tool',
              'inputSchema': {'type': 'object'}
            }
          ]
        }
      }) + '\n');

      final tools = await future;
      expect(tools.length, 1);
      expect(tools.first.name, 'test_tool');
    });

    test('callTool returns result', () async {
      await client.start();

      final future = client.callTool('test', {});
      await Future.delayed(Duration.zero);

      final request = jsonDecode(mockProcess.writtenLines.first);
      expect(request['method'], 'tools/call');

      mockProcess.emitStdout(jsonEncode({
        'jsonrpc': '2.0',
        'id': request['id'],
        'result': {
          'content': [{'type': 'text', 'text': 'result'}]
        }
      }) + '\n');

      final result = await future;
      expect(result.isError, false);
      expect(result.content, [{'type': 'text', 'text': 'result'}]);
    });

    test('handles errors', () async {
       await client.start();

       final future = client.listTools();
       await Future.delayed(Duration.zero);

       final request = jsonDecode(mockProcess.writtenLines.first);

       mockProcess.emitStdout(jsonEncode({
         'jsonrpc': '2.0',
         'id': request['id'],
         'error': {'code': -32000, 'message': 'Internal error'}
       }) + '\n');

       expect(future, throwsA(isA<Map<String, dynamic>>()));
    });
  });
}
