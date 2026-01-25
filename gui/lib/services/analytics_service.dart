// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'dart:math';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

class AnalyticsService {
  // Defaults (fallback)
  String _measurementId = 'G-7QZ03P9QGC';
  String _apiSecret = '2aYPyiPoQBiOuvhY7ad01Q';

  // Remote Config URL (Raw GitHub File)
  static const String _configUrl = 'https://raw.githubusercontent.com/agntcy/dir/main/gui/analytics_config.json';

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
      final prefs = await SharedPreferences.getInstance();
      _clientId = prefs.getString('ga_client_id');

      if (_clientId == null) {
        _clientId = _generateRandomId();
        await prefs.setString('ga_client_id', _clientId!);
      }

      // Generate a new session ID for this app run
      // GA4 sessions are usually defined by a unique ID and a timestamp
      _sessionId = DateTime.now().millisecondsSinceEpoch.toString();

      // Attempt to fetch latest config from GitHub
      await _fetchRemoteConfig();

      if (kDebugMode) {
        print('Analytics Initialized. ClientID: $_clientId, SessionID: $_sessionId');
      }
    } catch (e) {
      if (kDebugMode) print('Failed to init analytics: $e');
    }
  }

  /// Fetches the analytics configuration from the GitHub repository.
  /// Allows dynamic updates of Measurement ID and API Secret without app updates.
  Future<void> _fetchRemoteConfig() async {
    try {
      final response = await _client.get(Uri.parse(_configUrl));
      if (response.statusCode == 200) {
        final config = jsonDecode(response.body);
        if (config is Map) {
          if (config['measurement_id'] != null) {
             _measurementId = config['measurement_id'];
          }
          if (config['api_secret'] != null) {
             _apiSecret = config['api_secret'];
          }
           if (kDebugMode) print('Analytics Config Updated from Remote: $_measurementId');
        }
      }
    } catch (e) {
      if (kDebugMode) print('Failed to fetch analytics config: $e');
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
      // specialized logging for non-web platforms via http
      // Fire and forget - don't await full response in critical path if unimportant
      _client.post(
        uri,
        headers: {'Content-Type': 'application/json'},
        body: body,
      ).then((response) {
        if (kDebugMode && response.statusCode >= 300) {
           print('Analytics Error [${response.statusCode}]: ${response.body}');
        }
      });
    } catch (e) {
      if (kDebugMode) print('Analytics exception: $e');
    }
  }

  /// Generate a pseudo-random ID for client identification
  String _generateRandomId() {
    final rnd = Random();
    // Generate 16 bytes of random hex
    return List.generate(32, (i) => rnd.nextInt(16).toRadixString(16)).join();
  }
}
