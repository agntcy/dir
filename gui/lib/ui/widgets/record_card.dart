// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class RecordGrid extends StatelessWidget {
  final List<dynamic> items;
  final String? source;

  const RecordGrid({super.key, required this.items, this.source});

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) return const SizedBox.shrink();

    // Heuristic: If items are simple maps, use grid.
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (source != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8.0, left: 4),
            child: Text(
              'Results from $source (${items.length})',
              style: const TextStyle(fontWeight: FontWeight.bold, color: Colors.grey),
            ),
          ),
        GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 2,
            childAspectRatio: 1.2,
            crossAxisSpacing: 8,
            mainAxisSpacing: 8,
          ),
          itemCount: items.length,
          itemBuilder: (context, index) {
            final item = items[index];
            if (item is Map) {
              return RecordCard(data: Map<String, dynamic>.from(item), compact: true);
            }
            return Card(child: Center(child: Text(item.toString())));
          },
        ),
      ],
    );
  }
}

class RecordCard extends StatelessWidget {
  final Map<String, dynamic> data;
  final String? title;
  final bool compact;

  const RecordCard({
    super.key,
    required this.data,
    this.title,
    this.compact = false,
  });

  @override
  Widget build(BuildContext context) {
    // 1. Identify "Type" of record for summarization
    final isSearchResults = data.containsKey('record_cids') && data.containsKey('count');
    final isAgentRecord = data.containsKey('data') && data['data'] is Map;
    final isVersionList = data.containsKey('versions');

    Color baseColor = Colors.blue;
    IconData icon = Icons.data_object;
    String displayTitle = title ?? 'Data';

    if (isSearchResults) {
      baseColor = Colors.orange;
      icon = Icons.search;
      displayTitle = 'Search Results';
    } else if (isAgentRecord) {
      baseColor = Colors.purple;
      icon = Icons.smart_toy;
      displayTitle = _extractName() ?? 'Agent Record';
    }

    return Container(
      margin: compact ? EdgeInsets.zero : const EdgeInsets.symmetric(vertical: 8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border.all(color: baseColor.withOpacity(0.3)),
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: baseColor.withOpacity(0.05),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Header
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: baseColor.withOpacity(0.1),
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(11),
                topRight: const Radius.circular(11),
                bottomLeft: compact ? const Radius.circular(11) : Radius.zero,
                bottomRight: compact ? const Radius.circular(11) : Radius.zero,
              ),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Row(
                    children: [
                      Icon(icon, size: 16, color: baseColor),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          displayTitle,
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 13,
                            color: baseColor.withOpacity(0.8),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                if (!compact)
                  IconButton(
                    icon: Icon(Icons.copy, size: 16, color: baseColor),
                    tooltip: 'Copy JSON',
                    onPressed: () {
                      Clipboard.setData(ClipboardData(text: const JsonEncoder.withIndent('  ').convert(data)));
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Copied into clipboard')),
                      );
                    },
                  ),
              ],
            ),
          ),

          // Body (Hide if compact and header covers it? No, show minimal)
          Expanded(
            flex: compact ? 1 : 0,
            child: Padding(
              padding: const EdgeInsets.all(12.0),
              child: _buildContent(context, isSearchResults, isAgentRecord),
            ),
          ),
        ],
      ),
    );
  }

  String? _extractName() {
    if (data.containsKey('data')) {
      final d = data['data'];
      if (d is Map) {
        if (d.containsKey('name')) return d['name'];
        if (d.containsKey('caption')) return d['caption'];
      }
    }
    return null;
  }

  Widget _buildContent(BuildContext context, bool isSearch, bool isRecord) {
    if (isSearch) {
      return _buildSearchSummary();
    }
    if (isRecord) {
      return _buildRecordSummary(context);
    }
    return _buildJsonViewer(data);
  }

  Widget _buildSearchSummary() {
    final count = data['count'] ?? 0;
    final cids = data['record_cids'] as List?;

    if (compact) {
      return Center(child: _statBadge('Found', count.toString(), Colors.orange));
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _statBadge('Count', count.toString(), Colors.orange),
            _statBadge('More?', (data['has_more'] == true).toString(), Colors.grey),
          ],
        ),
        const SizedBox(height: 12),
        if (cids != null && cids.isNotEmpty) ...[
          const Text('CIDs Found:', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
          const SizedBox(height: 4),
          Wrap(
            spacing: 4,
            runSpacing: 4,
            children: cids.map((c) => Chip(
              label: Text(c.toString().substring(0, 8) + '...', style: const TextStyle(fontSize: 10)),
              backgroundColor: Colors.orange[50],
              padding: EdgeInsets.zero,
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
            )).toList(),
          ),
        ] else
          const Text('No records found.', style: TextStyle(fontStyle: FontStyle.italic, color: Colors.grey)),

        if (data.containsKey('error_message') && data['error_message'].toString().isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 8.0),
            child: Text(
              'Error: ${data['error_message']}',
              style: const TextStyle(color: Colors.red, fontSize: 11),
            ),
          ),
      ],
    );
  }

  Widget _buildRecordSummary(BuildContext context) {
    final d = data['data'] as Map;
    // Heuristics for common OASF fields
    final caption = d['caption'] ?? d['name'] ?? 'Untitled';
    final description = d['description'] ?? 'No description provided';
    final type = d['type'] ?? 'Unknown Type';
    final version = d['version'] ?? 'v0.0.1';
    final author = d['author'] ?? 'Unknown Author';

    if (compact) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(color: Colors.purple.shade50, borderRadius: BorderRadius.circular(8)),
                child: const Icon(Icons.smart_toy, color: Colors.purple, size: 18),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(caption, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12)),
                    Text(type, style: TextStyle(fontSize: 10, color: Colors.grey[600])),
                  ],
                ),
              )
            ],
          ),
          const SizedBox(height: 8),
          Expanded(
            child: Text(
              description,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 11),
            ),
          ),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
               _tag(version, Colors.blueGrey),
               Icon(Icons.arrow_forward, size: 14, color: Colors.grey[300]),
            ],
          )
        ],
      );
    }

    // Full View
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
           crossAxisAlignment: CrossAxisAlignment.start,
           children: [
             Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(color: Colors.purple.withOpacity(0.1), borderRadius: BorderRadius.circular(12)),
                child: const Icon(Icons.smart_toy, color: Colors.purple, size: 24),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(caption, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        _tag(type.toString(), Colors.purple),
                        const SizedBox(width: 8),
                        _tag(version.toString(), Colors.blueGrey),
                        const SizedBox(width: 8),
                        Icon(Icons.person, size: 14, color: Colors.grey[600]),
                        const SizedBox(width: 4),
                        Text(author, style: TextStyle(fontSize: 12, color: Colors.grey[600])),
                      ],
                    ),
                  ],
                ),
              ),
           ],
        ),
        const SizedBox(height: 16),
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(color: Theme.of(context).colorScheme.surfaceContainerHighest.withOpacity(0.3), borderRadius: BorderRadius.circular(8)),
          child: Text(description, style: const TextStyle(fontSize: 14, height: 1.4)),
        ),
        const SizedBox(height: 12),
        _buildGraphicalMetadata(d, context),
      ],
    );
  }

  Widget _buildGraphicalMetadata(Map d, BuildContext context) {
    final tags = d['tags'];
    final license = d['license'];
    final url = d['homepage'] ?? d['repository'];

    if (tags == null && license == null && url == null) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (tags is List && tags.isNotEmpty)
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: tags.map((t) => Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.secondaryContainer.withOpacity(0.4),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: Theme.of(context).colorScheme.outline.withOpacity(0.3)),
              ),
              child: Text(
                t.toString(),
                style: TextStyle(fontSize: 11, color: Theme.of(context).colorScheme.onSecondaryContainer),
              ),
            )).toList(),
          ),

        if (tags is List && tags.isNotEmpty) const SizedBox(height: 12),

        if (license != null || url != null)
           Row(
             children: [
               if (license != null) ...[
                 Icon(Icons.balance, size: 14, color: Theme.of(context).colorScheme.onSurfaceVariant),
                 const SizedBox(width: 4),
                 Text(license.toString(), style: TextStyle(fontSize: 12, color: Theme.of(context).colorScheme.onSurface)),
                 const SizedBox(width: 16),
               ],
               if (url != null) ...[
                 Icon(Icons.link, size: 14, color: Theme.of(context).colorScheme.primary),
                 const SizedBox(width: 4),
                 Expanded(child: Text(url.toString(), maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyle(fontSize: 12, color: Theme.of(context).colorScheme.primary, decoration: TextDecoration.underline))),
               ]
             ],
           )
      ],
    );
  }

  Widget _buildJsonViewer(Map<String, dynamic> json) {
    if (json.isEmpty) {
      return const Text('Empty', style: TextStyle(fontStyle: FontStyle.italic));
    }
    // Limit for compact
    final entries = compact ? json.entries.take(3) : json.entries;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: entries.map((e) => _buildField(e.key, e.value)).toList(),
    );
  }

  Widget _buildField(String key, dynamic value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 2.0),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '$key: ',
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 11),
          ),
          Expanded(
            child: Text(
              value.toString(),
              maxLines: compact ? 1 : 10,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 11, fontFamily: 'Courier'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _statBadge(String label, String value, Color color) {
    return Column(
      children: [
        Text(value, style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: color)),
        Text(label, style: const TextStyle(fontSize: 10, color: Colors.grey)),
      ],
    );
  }

  Widget _tag(String label, Color color) {
     return Container(
       padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
       decoration: BoxDecoration(
         color: color.withOpacity(0.1),
         borderRadius: BorderRadius.circular(4),
         border: Border.all(color: color.withOpacity(0.2)),
       ),
       child: Text(
         label,
         style: TextStyle(fontSize: 10, color: color, fontWeight: FontWeight.bold),
       ),
     );
  }
}
