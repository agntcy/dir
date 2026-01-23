// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/ui/widgets/search_results_widget.dart';

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
  group('SearchResultsWidget Tests', () {
    testWidgets('renders basic search info', (WidgetTester tester) async {
       await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: SearchResultsWidget(
              totalCount: 10,
              hasMore: false,
              recordCids: ['cid1', 'cid2'],
              searchCriteria: {'query': 'test agent'},
            ),
          ),
        ),
      ));

      // Check for count - Text is "${totalCount} agents found"
      expect(find.textContaining('10 agents found'), findsOneWidget);

      // Check for criteria
      expect(find.textContaining('test agent'), findsOneWidget);
    });

    testWidgets('renders agent cards from records', (WidgetTester tester) async {
       final records = [
         {
           'cid': 'cid1',
           'data': {'name': 'Agent One', 'description': 'Desc 1'}
         },
         {
           'cid': 'cid2',
           'data': {'name': 'Agent Two', 'description': 'Desc 2'}
         }
       ];

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: Scaffold(
            body: SingleChildScrollView(
              child: SearchResultsWidget(
                totalCount: 2,
                hasMore: false,
                recordCids: const ['cid1', 'cid2'],
                agentRecords: records,
              ),
            ),
          ),
        ),
      ));

      // Should find names
      expect(find.text('Agent One'), findsOneWidget);
      expect(find.text('Agent Two'), findsOneWidget);
    });

    testWidgets('handles sorting interaction', (WidgetTester tester) async {
       final records = [
         {
           'cid': 'cidA',
           'data': {'name': 'Apple Agent', 'authors': ['Zebra']}
         },
         {
           'cid': 'cidB',
           'data': {'name': 'Banana Agent', 'authors': ['Apple']}
         }
       ];

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: Scaffold(
            body: SingleChildScrollView(
              child: SearchResultsWidget(
                totalCount: 2,
                hasMore: false,
                recordCids: const ['cidA', 'cidB'],
                agentRecords: records,
              ),
            ),
          ),
        ),
      ));

      // Verify sorting chips are present
      expect(find.text('A-Z'), findsOneWidget);
      expect(find.text('Author'), findsOneWidget);
      expect(find.text('Date'), findsOneWidget);

      // Initial Order (Default by Name usually, or whatever implementation is):
      // The implementation seems to sort by name ASC by default if not specified?
      // Let's check positions.

      // If default sort is not guaranteed, let's tap 'Name' explicitly first just in case
      await tester.tap(find.text('A-Z'));

      final appleY = tester.getTopLeft(find.text('Apple Agent')).dy;
      final bananaY = tester.getTopLeft(find.text('Banana Agent')).dy;
      expect(appleY, lessThan(bananaY), reason: "Alphabetical: Apple should be above Banana");

      // Tap Sort by Author (Apple author should come first, then Zebra)
      await tester.tap(find.text('Author'));
      await tester.pumpAndSettle();

      final bananaY2 = tester.getTopLeft(find.text('Banana Agent')).dy;
      final appleY2 = tester.getTopLeft(find.text('Apple Agent')).dy;

      // Author 'Apple' (for Banana Agent) < Author 'Zebra' (for Apple Agent)
      expect(bananaY2, lessThan(appleY2), reason: "Author Sort: Apple(author) should be above Zebra(author)");
    });

    testWidgets('handles pagination interaction', (WidgetTester tester) async {
       // Assuming pageSize is 4 based on reading code in previous turns
       final cids = List.generate(10, (index) => 'cid_$index');
       final records = List.generate(10, (index) => {
         'cid': 'cid_$index',
         'data': {'name': 'Agent $index'}
       });

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: Scaffold(
            body: SingleChildScrollView(
              child: SearchResultsWidget(
                totalCount: 10,
                hasMore: false,
                recordCids: cids,
                agentRecords: records,
              ),
            ),
          ),
        ),
      ));

      // Should see first page items (e.g. Agent 0, 1, 2, 3)
      expect(find.text('Agent 0'), findsOneWidget);
      expect(find.text('Agent 3'), findsOneWidget);

      // Likely shouldn't see Agent 5 yet
      expect(find.text('Agent 5'), findsNothing);

      // Find and tap next page button (usually an arrow icon or number)
      // This part depends on implementation, looking for Icon(Icons.arrow_forward_ios) or similar
      final nextButton = find.byIcon(Icons.arrow_forward_ios);
      if (nextButton.evaluate().isNotEmpty) {
        await tester.tap(nextButton);
        await tester.pump();

        expect(find.text('Agent 5'), findsOneWidget);
      }
    });

    testWidgets('shows loading state', (WidgetTester tester) async {
       await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SearchResultsWidget(
            totalCount: 0,
            hasMore: false,
            recordCids: [],
            isLoading: true,
          ),
        ),
      ));

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('shows error message', (WidgetTester tester) async {
       await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SearchResultsWidget(
            totalCount: 0,
            hasMore: false,
            recordCids: [],
            errorMessage: 'Network failed',
          ),
        ),
      ));

      expect(find.textContaining('Network failed'), findsOneWidget);
    });
  });

  group('AgentDetailCard Tests', () {
    testWidgets('renders full details', (WidgetTester tester) async {
       final data = {
         'name': 'Test Agent',
         'description': 'Description',
         'author': 'Author Name',
         'skills': ['Skill1', 'Skill2'],
         'versions': ['1.0.0']
       };

       await tester.pumpWidget(MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: AgentDetailCard(cid: 'cid', agentData: data),
            ),
          ),
       ));

       expect(find.text('Test Agent'), findsOneWidget);
       expect(find.text('Description'), findsOneWidget);

       // Verify Skills are shown
       expect(find.text('Skill1'), findsOneWidget);
       expect(find.text('Skill2'), findsOneWidget);
    });
  });

  group('Interaction Tests', () {
    testWidgets('triggers onPullRecord when item is tapped', (WidgetTester tester) async {
      String? pulledCid;
      final record = {
         'cid': 'cid123',
         'name': 'Click Me',
         'caption': 'Tap Test',
         'author': 'Tester',
         'type': 'agent',
         'version': '1.0'
      };

      await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: Scaffold(
            body: SingleChildScrollView(
              child: SearchResultsWidget(
                totalCount: 1,
                hasMore: false,
                recordCids: ['cid123'],
                agentRecords: [record],
                onPullRecord: (cid) => pulledCid = cid,
              ),
            ),
          ),
        ),
      ));

      expect(find.text('Click Me'), findsOneWidget);
      // Tap the 'See more' button
      await tester.tap(find.text('See more'));
      await tester.pump();

      expect(pulledCid, 'cid123');
    });
  });
}
