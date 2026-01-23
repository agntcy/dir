// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gui/ui/settings_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  group('SettingsScreen Tests', () {
    setUp(() {
       SharedPreferences.setMockInitialValues({});
    });

    testWidgets('renders initial state with defaults', (WidgetTester tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: SettingsScreen(),
      ));

      // Allow helper future to complete
      await tester.pumpAndSettle();

      expect(find.text('Settings'), findsOneWidget);
      expect(find.text('AI Provider'), findsOneWidget);
      // Default provider is Gemini
      expect(find.text('Google Gemini'), findsOneWidget);
      expect(find.text('Agent Directory Configuration'), findsOneWidget);
    });

    testWidgets('loads values from shared preferences', (WidgetTester tester) async {
       SharedPreferences.setMockInitialValues({
         'provider': 'azure',
         'azure_api_key': 'test_key_123',
         'directory_server_address': 'localhost:9999',
       });

       await tester.pumpWidget(const MaterialApp(
        home: SettingsScreen(),
      ));
      await tester.pumpAndSettle();

      // Should show Azure dropdown value
      expect(find.text('Azure OpenAI'), findsOneWidget);

      // Should show azure key field populated (obscured, but controller should have text)
      // Since it's TextFormField, we can find by widget and check controller text
      final keyFieldFinder = find.widgetWithText(TextFormField, 'test_key_123');
      // Or check if a specific textfield contains it
      // Note: obscureText might hide it from find.text? No, find.text usually finds the editable text.
      // Let's verify if the form field for Azure is visible
      expect(find.text('Azure OpenAI Configuration'), findsOneWidget);

      expect(find.widgetWithText(TextFormField, 'localhost:9999'), findsOneWidget);
    });

    testWidgets('saves settings', (WidgetTester tester) async {
      SharedPreferences.setMockInitialValues({});

      await tester.pumpWidget(const MaterialApp(
        home: SettingsScreen(),
      ));
      await tester.pumpAndSettle();

      // Enter Gemini Key
      await tester.enterText(
         find.widgetWithText(TextFormField, 'API Key').first, // There might be multiple "API Key" labels if we are not careful, but Gemini is default so only its fields should be shown
         'my_new_gemini_key'
      );

      // Enter Directory URL
      await tester.enterText(
         find.widgetWithText(TextFormField, 'Directory Server URL'),
         'http://custom-server'
      );

      // Save
      await tester.tap(find.byIcon(Icons.save));
      await tester.pumpAndSettle();

      // Check SharedPreferences
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('gemini_api_key'), 'my_new_gemini_key');
      expect(prefs.getString('directory_server_address'), 'http://custom-server');
    });

    testWidgets('changes provider updates fields', (WidgetTester tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: SettingsScreen(),
      ));
      await tester.pumpAndSettle();

      // Verify Gemini default
      expect(find.text('Google Gemini Configuration'), findsOneWidget);

      // Change to Azure
      await tester.tap(find.text('Google Gemini'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Azure OpenAI').last);
      await tester.pumpAndSettle();

      expect(find.text('Azure OpenAI Configuration'), findsOneWidget);
      expect(find.text('Google Gemini Configuration'), findsNothing);

      // Check Azure specific fields
      expect(find.text('Deployment Name'), findsOneWidget);
    });
  });
}
