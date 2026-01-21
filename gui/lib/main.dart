// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter/material.dart';

import 'ui/chat_screen.dart';

void main() {
  runApp(const MyApp());
}

// AGNTCY Brand Colors (from blogs.agntcy.org)
class AgntcyColors {
  // Light theme
  static const Color lightAccent = Color(0xFF187ADC);      // Blue accent
  static const Color lightSurface = Color(0xFFEFF3FC);     // Light blue-gray surface
  static const Color lightBackground = Color(0xFFFFFFFF);  // White background
  static const Color lightTextPrimary = Color(0xFF1C1E21); // Dark text
  static const Color lightTextSecondary = Color(0xFF828282); // Gray text
  
  // Dark theme  
  static const Color darkAccent = Color(0xFFFBAF45);       // Orange/amber accent
  static const Color darkSurface = Color(0xFF04142B);      // Very dark navy surface
  static const Color darkBackground = Color(0xFF020B18);   // Even darker background
  static const Color darkTextPrimary = Color(0xFFE3E3E3);  // Light text
  static const Color darkTextSecondary = Color(0xFF9CA3AF); // Muted text
}

class MyApp extends StatefulWidget {
  const MyApp({super.key});

  static void toggleTheme(BuildContext context) {
    context.findAncestorStateOfType<_MyAppState>()?.toggleTheme();
  }

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  ThemeMode _themeMode = ThemeMode.light;

  void toggleTheme() {
    setState(() {
      _themeMode = _themeMode == ThemeMode.light ? ThemeMode.dark : ThemeMode.light;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'AGNTCY Agent Directory GUI',
      theme: ThemeData(
        brightness: Brightness.light,
        scaffoldBackgroundColor: AgntcyColors.lightBackground,
        colorScheme: ColorScheme.light(
          primary: AgntcyColors.lightAccent,
          secondary: AgntcyColors.lightAccent.withOpacity(0.8),
          surface: AgntcyColors.lightSurface,
          onPrimary: Colors.white,
          onSecondary: Colors.white,
          onSurface: AgntcyColors.lightTextPrimary,
        ),
        appBarTheme: const AppBarTheme(
          backgroundColor: AgntcyColors.lightSurface,
          foregroundColor: AgntcyColors.lightTextPrimary,
          elevation: 1,
        ),
        inputDecorationTheme: InputDecorationTheme(
          filled: true,
          fillColor: Colors.white,
          hintStyle: const TextStyle(color: AgntcyColors.lightTextSecondary),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: BorderSide(color: AgntcyColors.lightAccent.withOpacity(0.3)),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: BorderSide(color: AgntcyColors.lightAccent.withOpacity(0.3)),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: const BorderSide(color: AgntcyColors.lightAccent, width: 2),
          ),
        ),
        cardTheme: CardThemeData(
          color: Colors.white,
          elevation: 2,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        ),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        brightness: Brightness.dark,
        scaffoldBackgroundColor: AgntcyColors.darkBackground,
        colorScheme: ColorScheme.dark(
          primary: AgntcyColors.darkAccent,
          secondary: AgntcyColors.darkAccent.withOpacity(0.8),
          surface: AgntcyColors.darkSurface,
          onPrimary: AgntcyColors.darkSurface,
          onSecondary: AgntcyColors.darkSurface,
          onSurface: AgntcyColors.darkTextPrimary,
        ),
        appBarTheme: const AppBarTheme(
          backgroundColor: AgntcyColors.darkSurface,
          foregroundColor: AgntcyColors.darkTextPrimary,
          elevation: 0,
        ),
        inputDecorationTheme: InputDecorationTheme(
          filled: true,
          fillColor: AgntcyColors.darkSurface,
          hintStyle: const TextStyle(color: AgntcyColors.darkTextSecondary),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: BorderSide(color: AgntcyColors.darkAccent.withOpacity(0.3)),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: BorderSide(color: AgntcyColors.darkAccent.withOpacity(0.4)),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
            borderSide: const BorderSide(color: AgntcyColors.darkAccent, width: 2),
          ),
        ),
        cardTheme: CardThemeData(
          color: AgntcyColors.darkSurface,
          elevation: 2,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        ),
        useMaterial3: true,
      ),
      themeMode: _themeMode,
      home: const ChatScreen(),
      debugShowCheckedModeBanner: false,
    );
  }
}
