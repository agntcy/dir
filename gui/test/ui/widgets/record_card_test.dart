// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/ui/widgets/record_card.dart';

// Mock AssetBundle for SVGs
class MockAssetBundle extends CachingAssetBundle {
  @override
  Future<String> loadString(String key, {bool cache = true}) async {
    return '<svg viewBox="0 0 1 1"></svg>';
  }

  @override
  Future<ByteData> load(String key) async {
    return ByteData.view(Uint8List.fromList('<svg viewBox="0 0 1 1"></svg>'.codeUnits).buffer);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  // Helper to pump widget
  Future<void> pumpCard(WidgetTester tester, Map<String, dynamic> data, {bool compact = false}) async {
    await tester.pumpWidget(MaterialApp(
      home: DefaultAssetBundle(
        bundle: MockAssetBundle(),
        child: Scaffold(
          body: RecordCard(data: data, compact: compact),
        ),
      ),
    ));
  }

  group('RecordGrid Tests', () {
    testWidgets('renders grid of RecordCards', (WidgetTester tester) async {
      final items = [
        {'name': 'Item A'},
        {'name': 'Item B'},
      ];

      await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: Scaffold(
            body: SingleChildScrollView(
              child: RecordGrid(items: items, source: 'TestSource')
            ),
          ),
        ),
      ));

      expect(find.text('Results from TestSource (2)'), findsOneWidget);
      expect(find.text('Item A'), findsOneWidget);
      expect(find.text('Item B'), findsOneWidget);
    });

    testWidgets('renders non-map items as simple cards', (WidgetTester tester) async {
      final items = ['Simple String'];

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: RecordGrid(items: items),
        ),
      ));

      expect(find.text('Simple String'), findsOneWidget);
    });
  });

  group('RecordCard Tests', () {
    testWidgets('renders basic map data', (WidgetTester tester) async {
      final data = {
        'name': 'Test Name',
        'key1': 'value1',
      };
      await pumpCard(tester, data);
      expect(find.text('Data'), findsOneWidget); // Default title
      expect(find.textContaining('key1'), findsOneWidget); // Json viewer content
      expect(find.textContaining('value1'), findsOneWidget);
    });

    testWidgets('identifies Search Results (Compact)', (WidgetTester tester) async {
      final data = {
        'count': 5,
        'record_cids': ['aaaabbbbcccc', '111122223333'],
      };
      await pumpCard(tester, data, compact: true);

      expect(find.text('Search Results'), findsOneWidget);
      expect(find.text('Found'), findsOneWidget);
      expect(find.text('5'), findsOneWidget);
    });

    testWidgets('identifies Search Results (Full)', (WidgetTester tester) async {
      final data = {
        'count': 10,
        'record_cids': ['aaaabbbbcccc', '111122223333'],
      };
      await pumpCard(tester, data, compact: false);

      expect(find.text('Count'), findsOneWidget);
      expect(find.text('10'), findsOneWidget);
    });

    testWidgets('identifies Agent Record and extracts name', (WidgetTester tester) async {
      final data = {
        'data': {
          'name': 'Agent 007',
          'type': 'spy'
        }
      };
      await pumpCard(tester, data);

      expect(find.text('Agent 007'), findsWidgets); // Extracted title (header + body)
      expect(find.byIcon(Icons.smart_toy), findsWidgets);
    });

    testWidgets('identifies Agent Record and extracts caption if name missing', (WidgetTester tester) async {
      final data = {
        'data': {
          'caption': 'Secret Agent',
        }
      };
      await pumpCard(tester, data);

      expect(find.text('Secret Agent'), findsWidgets);
    });

    testWidgets('Copy Button functionality', (WidgetTester tester) async {
      final data = {'test': 123};

      // Intercept system channels or just verify call
      final log = <MethodCall>[];
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        (MethodCall methodCall) async {
          if (methodCall.method == 'Clipboard.setData') {
            log.add(methodCall);
          }
          return null;
        },
      );

      await pumpCard(tester, data, compact: false);

      // Tap copy
      await tester.tap(find.byIcon(Icons.copy));
      await tester.pump(); // frame for snackbar handling

      expect(log, isNotEmpty);
      expect(log.last.method, 'Clipboard.setData');
      final content = (log.last.arguments as Map)['text'];
      expect(content, contains('"test": 123'));

      // Verify SnackBar
      expect(find.text('Copied into clipboard'), findsOneWidget);
    });
  });
}
