// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _formKey = GlobalKey<FormState>();

  // Provider Selection
  String _selectedProvider = 'gemini';

  // Controllers
  final _geminiKeyController = TextEditingController();

  final _azureKeyController = TextEditingController();
  final _azureEndpointController = TextEditingController();
  final _azureDeploymentController = TextEditingController();

  final _openaiKeyController = TextEditingController();
  final _openaiEndpointController = TextEditingController();

  // Directory Configuration
  final _directoryUrlController = TextEditingController();
  final _directoryTokenController = TextEditingController();
  final _oasfSchemaController = TextEditingController();

  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadSettings();
  }

  Future<void> _loadSettings() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _selectedProvider = prefs.getString('provider') ?? 'gemini';

      _geminiKeyController.text = prefs.getString('gemini_api_key') ?? '';

      _azureKeyController.text = prefs.getString('azure_api_key') ?? '';
      _azureEndpointController.text = prefs.getString('azure_endpoint') ?? '';
      _azureDeploymentController.text = prefs.getString('azure_deployment') ?? '';

      _openaiKeyController.text = prefs.getString('openai_api_key') ?? '';
      _openaiEndpointController.text = prefs.getString('openai_endpoint') ?? '';

      _directoryUrlController.text = prefs.getString('directory_server_address') ?? '';
      _directoryTokenController.text = prefs.getString('directory_github_token') ?? '';
      _oasfSchemaController.text = prefs.getString('oasf_schema_url') ?? '';

      _isLoading = false;
    });
  }

  Future<void> _saveSettings() async {
    if (!_formKey.currentState!.validate()) return;

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('provider', _selectedProvider);

    await prefs.setString('gemini_api_key', _geminiKeyController.text.trim());

    await prefs.setString('azure_api_key', _azureKeyController.text.trim());
    await prefs.setString('azure_endpoint', _azureEndpointController.text.trim());
    await prefs.setString('azure_deployment', _azureDeploymentController.text.trim());

    await prefs.setString('openai_api_key', _openaiKeyController.text.trim());
    await prefs.setString('openai_endpoint', _openaiEndpointController.text.trim());

    await prefs.setString('directory_server_address', _directoryUrlController.text.trim());
    await prefs.setString('directory_github_token', _directoryTokenController.text.trim());
    await prefs.setString('oasf_schema_url', _oasfSchemaController.text.trim());

    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Settings saved')),
      );
      Navigator.pop(context, true); // Return true to indicate settings changed
    }
  }

  @override
  void dispose() {
    _geminiKeyController.dispose();
    _azureKeyController.dispose();
    _azureEndpointController.dispose();
    _azureDeploymentController.dispose();
    _openaiKeyController.dispose();
    _openaiEndpointController.dispose();

    _directoryUrlController.dispose();
    _directoryTokenController.dispose();
    _oasfSchemaController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            onPressed: _saveSettings,
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16.0),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('AI Provider', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              const SizedBox(height: 8),
              DropdownButtonFormField<String>(
                value: _selectedProvider,
                decoration: const InputDecoration(
                  border: OutlineInputBorder(),
                  labelText: 'Select Provider',
                ),
                items: const [
                  DropdownMenuItem(value: 'gemini', child: Text('Google Gemini')),
                  DropdownMenuItem(value: 'azure', child: Text('Azure OpenAI')),
                  DropdownMenuItem(value: 'openai', child: Text('OpenAI Compatible (Claude, etc.)')),
                ],
                onChanged: (value) {
                  setState(() {
                    _selectedProvider = value!;
                  });
                },
              ),
              const SizedBox(height: 24),

              if (_selectedProvider == 'gemini') ...[
                const Text('Google Gemini Configuration', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _geminiKeyController,
                  decoration: const InputDecoration(
                    labelText: 'API Key',
                    border: OutlineInputBorder(),
                    helperText: 'Get your key at aistudio.google.com',
                  ),
                  obscureText: true,
                  validator: (value) {
                    if (_selectedProvider == 'gemini' && (value == null || value.isEmpty)) {
                      return 'Please enter API Key';
                    }
                    return null;
                  },
                ),
              ],

              if (_selectedProvider == 'azure') ...[
                const Text('Azure OpenAI Configuration', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _azureKeyController,
                  decoration: const InputDecoration(
                    labelText: 'API Key',
                    border: OutlineInputBorder(),
                  ),
                  obscureText: true,
                  validator: (value) {
                    if (_selectedProvider == 'azure' && (value == null || value.isEmpty)) {
                      return 'Please enter API Key';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _azureEndpointController,
                  decoration: const InputDecoration(
                    labelText: 'Endpoint',
                    border: OutlineInputBorder(),
                    hintText: 'https://YOUR_RESOURCE_NAME.openai.azure.com/',
                  ),
                  validator: (value) {
                    if (_selectedProvider == 'azure' && (value == null || value.isEmpty)) {
                      return 'Please enter Endpoint';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _azureDeploymentController,
                  decoration: const InputDecoration(
                    labelText: 'Deployment Name',
                    border: OutlineInputBorder(),
                  ),
                  validator: (value) {
                    if (_selectedProvider == 'azure' && (value == null || value.isEmpty)) {
                      return 'Please enter Deployment Name';
                    }
                    return null;
                  },
                ),
              ],

              if (_selectedProvider == 'openai') ...[
                const Text('OpenAI-Compatible Configuration', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                const Text('Use this for OpenAI, Anthropic (via gateway), or local models (Ollama, vLLM).', style: TextStyle(color: Colors.grey)),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _openaiKeyController,
                  decoration: const InputDecoration(
                    labelText: 'API Key',
                    border: OutlineInputBorder(),
                  ),
                  obscureText: true,
                  validator: (value) {
                    if (_selectedProvider == 'openai' && (value == null || value.isEmpty)) {
                      return 'Please enter API Key';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _openaiEndpointController,
                  decoration: const InputDecoration(
                    labelText: 'Base URL',
                    border: OutlineInputBorder(),
                    hintText: 'https://api.openai.com/v1',
                  ),
                  validator: (value) {
                    if (_selectedProvider == 'openai' && (value == null || value.isEmpty)) {
                      return 'Please enter Base URL';
                    }
                    return null;
                  },
                ),
              ],

              const SizedBox(height: 32),
              const Divider(),
              const SizedBox(height: 16),

              const Text('Agent Directory Configuration', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              const SizedBox(height: 16),

              TextFormField(
                controller: _directoryUrlController,
                decoration: const InputDecoration(
                  labelText: 'Directory Server URL',
                  border: OutlineInputBorder(),
                  hintText: 'e.g. localhost:8888 or custom URL',
                  helperText: 'Leave empty for default (localhost:8888)',
                ),
              ),
              const SizedBox(height: 16),

              TextFormField(
                controller: _directoryTokenController,
                decoration: const InputDecoration(
                  labelText: 'Auth Token (Optional)',
                  border: OutlineInputBorder(),
                  helperText: 'GitHub PAT or other token if required',
                ),
                obscureText: true,
              ),
              const SizedBox(height: 16),

              TextFormField(
                controller: _oasfSchemaController,
                decoration: const InputDecoration(
                  labelText: 'OASF Schema URL (Optional)',
                  border: OutlineInputBorder(),
                  helperText: 'Custom validation schema URL',
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
