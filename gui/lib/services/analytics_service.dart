// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import '../env/env.dart';

class AnalyticsService {
  // Secured via Envied, but capable of runtime override
  String _measurementId = Env.measurementId;
  String _apiSecret = Env.apiSecret;

  static const String _logEndpoint = 'https://www.google-analytics.com/mp/collect';
  static const String _debugEndpoint = 'https://www.google-analytics.com/debug/mp/collect';

  final http.Client _client;
  String? _clientId;
  String? _sessionId;

  // Singleton instance
  static final AnalyticsService _instance = AnalyticsService._internal();
  factory AnalyticsService() => _instance;

  AnalyticsService._internal() : _client = http.Client();

  /// Initialize the analytics service.
  /// Generates or retrieves a persistent Client ID and creates a new Session ID.
  Future<void> init() async {
    try {
      // Runtime Overrides (useful for CI/CD or dev without rebuilding)
      if (Platform.environment.containsKey('MEASUREMENT_ID')) {
        _measurementId = Platform.environment['MEASUREMENT_ID']!;
      }
      if (Platform.environment.containsKey('GA_API_SECRET')) {
        _apiSecret = Platform.environment['GA_API_SECRET']!;
      }

      if (_measurementId.isEmpty || _apiSecret.isEmpty) {
        print('Analytics disabled: MEASUREMENT_ID or API_SECRET not configured.');
        return;
      }

      final prefs = await SharedPreferences.getInstance();
      _clientId = prefs.getString('ga_client_id');

      if (_clientId == null) {
        _clientId = _generateRandomId();
        await prefs.setString('ga_client_id', _clientId!);
      }

      // Generate a new session ID for this app run
      // GA4 sessions are usually defined by a unique ID and a timestamp
      _sessionId = DateTime.now().millisecondsSinceEpoch.toString();

      if (_measurementId == 'G-LOCALDEV00') {
         // If still default, warn
         print('WARNING: Analytics using placeholder MEASUREMENT_ID (G-LOCALDEV00). No data will be sent to real GA4.');
      }

      if (kDebugMode || _measurementId == 'G-LOCALDEV00') {
        print('Analytics Initialized. ClientID: $_clientId, SessionID: $_sessionId, MID: $_measurementId');
      }
    } catch (e) {
      if (kDebugMode) print('Failed to init analytics: $e');
    }
  }

  /// Log a custom event to Google Analytics 4
  Future<void> logEvent(String name, {Map<String, dynamic>? params}) async {
    if (_clientId == null) return; // Not initialized or opted out

    // Basic parameters required for session tracking
    final Map<String, dynamic> finalParams = {
      'session_id': _sessionId,
      'engagement_time_msec': 100, // Minimal engagement time to count as active
      ...?params,
    };

    final body = jsonEncode({
      'client_id': _clientId,
      'events': [
        {
          'name': name,
          'params': finalParams,
        }
      ]
    });

    final uri = Uri.parse('$_logEndpoint?measurement_id=$_measurementId&api_secret=$_apiSecret');

    try {
      print('GA4: Sending event "$name" ...');
      // specialized logging for non-web platforms via http
      // Fire and forget - don't await full response in critical path if unimportant
      _client.post(
        uri,
        headers: {'Content-Type': 'application/json'},
        body: body,
      ).then((response) {
        if (response.statusCode >= 200 && response.statusCode < 300) {
           print('GA4: Success [${response.statusCode}] for event "$name"');
        } else {
           print('GA4: Error [${response.statusCode}]: ${response.body}');
        }
      });
    } catch (e) {
      print('GA4: Exception: $e');
    }
  }

  /// Generate a pseudo-random ID for client identification
  String _generateRandomId() {
    final rnd = Random();
    // Generate 16 bytes of random hex
    return List.generate(32, (i) => rnd.nextInt(16).toRadixString(16)).join();
  }
}
