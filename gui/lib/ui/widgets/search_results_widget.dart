// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// A declarative search results widget displaying agent records
/// with pagination, search criteria reminder, and expandable cards.
class SearchResultsWidget extends StatefulWidget {
  final int totalCount;
  final bool hasMore;
  final List<String> recordCids;
  final Map<String, dynamic>? searchCriteria;
  final List<Map<String, dynamic>>? agentRecords;
  final String? errorMessage;
  final void Function(String cid)? onPullRecord;
  final void Function(int offset)? onLoadMore;

  const SearchResultsWidget({
    super.key,
    required this.totalCount,
    required this.hasMore,
    required this.recordCids,
    this.searchCriteria,
    this.agentRecords,
    this.errorMessage,
    this.onPullRecord,
    this.onLoadMore,
  });

  @override
  State<SearchResultsWidget> createState() => _SearchResultsWidgetState();
}

class _SearchResultsWidgetState extends State<SearchResultsWidget> {
  int _currentPage = 0;
  static const int _pageSize = 5;
  String _sortBy = 'default'; // 'default', 'name', 'date'


  List<String> get _sortedCids {
    if (widget.agentRecords == null || widget.agentRecords!.isEmpty) {
      return widget.recordCids;
    }
    
    // Create a list with CID and associated data for sorting
    final cidsWithData = widget.recordCids.map((cid) {
      Map<String, dynamic>? record;
      for (final r in widget.agentRecords!) {
        if (r['cid'] == cid) {
          record = r;
          break;
        }
      }
      return {'cid': cid, 'record': record};
    }).toList();
    
    // Sort based on selected option
    if (_sortBy == 'name') {
      cidsWithData.sort((a, b) {
        final aName = _extractName(a['record'] as Map<String, dynamic>?) ?? 'zzz';
        final bName = _extractName(b['record'] as Map<String, dynamic>?) ?? 'zzz';
        return aName.toLowerCase().compareTo(bName.toLowerCase());
      });
    } else if (_sortBy == 'date') {
      cidsWithData.sort((a, b) {
        final aDate = _extractDate(a['record'] as Map<String, dynamic>?);
        final bDate = _extractDate(b['record'] as Map<String, dynamic>?);
        // Sort by date descending (newest first)
        return bDate.compareTo(aDate);
      });
    }
    
    return cidsWithData.map((e) => e['cid'] as String).toList();
  }

  String? _extractName(Map<String, dynamic>? record) {
    if (record == null) return null;
    final data = record['data'] as Map<String, dynamic>?;
    return data?['name']?.toString() ?? record['name']?.toString();
  }

  String _extractDate(Map<String, dynamic>? record) {
    if (record == null) return '0000-00-00';
    final data = record['data'] as Map<String, dynamic>?;
    return data?['created_at']?.toString() ?? record['created_at']?.toString() ?? '0000-00-00';
  }

  List<String> get _paginatedCids {
    final sorted = _sortedCids;
    final start = _currentPage * _pageSize;
    final end = (start + _pageSize).clamp(0, sorted.length);
    return sorted.sublist(start, end);
  }

  int get _totalPages => (_sortedCids.length / _pageSize).ceil();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 12),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            colorScheme.primaryContainer.withOpacity(0.3),
            colorScheme.surface,
          ],
        ),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.primary.withOpacity(0.2),
          width: 1.5,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.primary.withOpacity(0.08),
            blurRadius: 12,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Header with count and criteria
          _buildHeader(context),
          
          // Error message if any
          if (widget.errorMessage != null && widget.errorMessage!.isNotEmpty)
            _buildErrorBanner(context),
          
          // Search criteria reminder
          if (widget.searchCriteria != null && widget.searchCriteria!.isNotEmpty)
            _buildSearchCriteriaReminder(context),
          
          // Divider line
          if (widget.searchCriteria != null && widget.searchCriteria!.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Divider(color: Theme.of(context).colorScheme.outline.withOpacity(0.2)),
            ),
          
          // Sort controls
          if (widget.recordCids.isNotEmpty)
            _buildSortControls(context),
          
          // Results list
          if (widget.recordCids.isNotEmpty) ...[
            _buildResultsList(context),
            _buildPaginationControls(context),
          ] else
            _buildEmptyState(context),
        ],
      ),
    );
  }

  Widget _buildHeader(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colorScheme.primary.withOpacity(0.1),
        borderRadius: const BorderRadius.vertical(top: Radius.circular(15)),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: colorScheme.primary.withOpacity(0.15),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(
              Icons.search_rounded,
              color: colorScheme.primary,
              size: 24,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Search Results',
                  style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.bold,
                    color: colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  '${widget.totalCount} agent${widget.totalCount == 1 ? '' : 's'} found${widget.hasMore ? ' (more available)' : ''}',
                  style: TextStyle(
                    fontSize: 13,
                    color: colorScheme.onSurface.withOpacity(0.7),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSearchCriteriaReminder(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final criteria = widget.searchCriteria!;
    
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Icon(
            Icons.filter_alt_outlined,
            size: 14,
            color: colorScheme.secondary,
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Wrap(
              spacing: 8,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: criteria.entries
                  .where((e) => e.value != null && e.value.toString().isNotEmpty)
                  .map((e) => Text(
                    '${e.key}: ${e.value}',
                    style: TextStyle(
                      fontSize: 12,
                      color: colorScheme.secondary,
                      fontStyle: FontStyle.italic,
                    ),
                  ))
                  .toList(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSortControls(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
      child: Row(
        children: [
          Text(
            'Sort by:',
            style: TextStyle(
              fontSize: 12,
              color: colorScheme.secondary,
            ),
          ),
          const SizedBox(width: 8),
          _buildSortChip(context, 'default', 'Default', Icons.sort),
          const SizedBox(width: 6),
          _buildSortChip(context, 'name', 'A-Z', Icons.sort_by_alpha),
          const SizedBox(width: 6),
          _buildSortChip(context, 'date', 'Date', Icons.calendar_today),
        ],
      ),
    );
  }

  Widget _buildSortChip(BuildContext context, String value, String label, IconData icon) {
    final colorScheme = Theme.of(context).colorScheme;
    final isSelected = _sortBy == value;
    
    return InkWell(
      onTap: () => setState(() => _sortBy = value),
      borderRadius: BorderRadius.circular(16),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
        decoration: BoxDecoration(
          color: isSelected 
              ? colorScheme.primary.withOpacity(0.15) 
              : colorScheme.surfaceContainerHighest.withOpacity(0.5),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isSelected 
                ? colorScheme.primary.withOpacity(0.4) 
                : colorScheme.outline.withOpacity(0.15),
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              icon,
              size: 12,
              // Grey only for inactive state
              color: isSelected 
                  ? colorScheme.primary 
                  : colorScheme.onSurface.withOpacity(0.4),
            ),
            const SizedBox(width: 4),
            Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: isSelected ? FontWeight.w600 : FontWeight.normal,
                // Grey only for inactive state
                color: isSelected 
                    ? colorScheme.primary 
                    : colorScheme.onSurface.withOpacity(0.4),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCriteriaChip(BuildContext context, String key, dynamic value) {
    final colorScheme = Theme.of(context).colorScheme;
    String displayValue = value is List ? value.join(', ') : value.toString();
    
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colorScheme.outline.withOpacity(0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$key: ',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: colorScheme.onSurface.withOpacity(0.6),
            ),
          ),
          Text(
            displayValue,
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w500,
              color: colorScheme.primary,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildResultsList(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Column(
        children: _paginatedCids.asMap().entries.map((entry) {
          final index = entry.key;
          final cid = entry.value;
          
          // Find matching agent record if available
          Map<String, dynamic>? agentRecord;
          if (widget.agentRecords != null) {
            for (final record in widget.agentRecords!) {
              if (record['cid'] == cid) {
                agentRecord = record;
                break;
              }
            }
          }
          
          return AgentRecordCard(
            cid: cid,
            agentData: agentRecord,
            index: _currentPage * _pageSize + index + 1,
            onPullDetails: widget.onPullRecord != null 
                ? () => widget.onPullRecord!(cid)
                : null,
          );
        }).toList(),
      ),
    );
  }

  Widget _buildPaginationControls(BuildContext context) {
    if (_totalPages <= 1) return const SizedBox(height: 12);
    
    final colorScheme = Theme.of(context).colorScheme;
    
    return Container(
      padding: const EdgeInsets.all(12),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          IconButton(
            onPressed: _currentPage > 0
                ? () => setState(() => _currentPage--)
                : null,
            icon: const Icon(Icons.chevron_left_rounded),
            style: IconButton.styleFrom(
              backgroundColor: _currentPage > 0
                  ? colorScheme.primaryContainer
                  : colorScheme.surfaceContainerHighest,
            ),
          ),
          const SizedBox(width: 12),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            decoration: BoxDecoration(
              color: colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(20),
            ),
            child: Text(
              'Page ${_currentPage + 1} of $_totalPages',
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: colorScheme.onSurface,
              ),
            ),
          ),
          const SizedBox(width: 12),
          IconButton(
            onPressed: _currentPage < _totalPages - 1
                ? () => setState(() => _currentPage++)
                : null,
            icon: const Icon(Icons.chevron_right_rounded),
            style: IconButton.styleFrom(
              backgroundColor: _currentPage < _totalPages - 1
                  ? colorScheme.primaryContainer
                  : colorScheme.surfaceContainerHighest,
            ),
          ),
          if (widget.hasMore) ...[
            const SizedBox(width: 16),
            TextButton.icon(
              onPressed: widget.onLoadMore != null
                  ? () => widget.onLoadMore!(widget.recordCids.length)
                  : null,
              icon: const Icon(Icons.add_circle_outline, size: 18),
              label: const Text('Load More'),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    
    return Container(
      padding: const EdgeInsets.all(32),
      child: Column(
        children: [
          Icon(
            Icons.search_off_rounded,
            size: 48,
            color: colorScheme.onSurface.withOpacity(0.3),
          ),
          const SizedBox(height: 12),
          Text(
            'No agents found',
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.w500,
              color: colorScheme.onSurface.withOpacity(0.6),
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Try adjusting your search criteria',
            style: TextStyle(
              fontSize: 13,
              color: colorScheme.onSurface.withOpacity(0.4),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildErrorBanner(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.red.withOpacity(0.1),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.red.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          const Icon(Icons.error_outline, color: Colors.red, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              widget.errorMessage!,
              style: const TextStyle(color: Colors.red, fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }
}

/// Agent card for list view - shows full content when loaded, placeholder when not
/// NOT expandable - Load button only shows when no data
class AgentRecordCard extends StatelessWidget {
  final String cid;
  final Map<String, dynamic>? agentData;
  final int index;
  final VoidCallback? onPullDetails;

  const AgentRecordCard({
    super.key,
    required this.cid,
    this.agentData,
    required this.index,
    this.onPullDetails,
  });

  Map<String, dynamic>? _extractAgentInfo() {
    final data = agentData;
    if (data == null) return null;
    if (data.containsKey('name') && data['name'] != null) {
      return data;
    } else if (data['data'] is Map) {
      return Map<String, dynamic>.from(data['data'] as Map);
    }
    return data;
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final agentInfo = _extractAgentInfo();
    
    final hasData = agentInfo != null && agentInfo.containsKey('name') && agentInfo['name'] != null;
    
    // Extract display values
    final rawName = hasData ? agentInfo!['name']?.toString() ?? '' : '';
    final name = hasData ? _extractDisplayName(rawName) : 'Loading agent...';
    final authors = agentInfo?['authors'];
    final author = authors is List && authors.isNotEmpty ? authors.first.toString() : '';
    final version = agentInfo?['version']?.toString() ?? '';
    
    final shortCid = cid.length > 16 
        ? '${cid.substring(0, 8)}...${cid.substring(cid.length - 4)}'
        : cid;
    
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: hasData ? colorScheme.primary.withOpacity(0.25) : colorScheme.outline.withOpacity(0.2),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.06),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            // Agent icon
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: hasData 
                    ? colorScheme.primaryContainer.withOpacity(0.4)
                    : colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(
                Icons.smart_toy_rounded,
                color: hasData ? colorScheme.primary : colorScheme.outline,
                size: 24,
              ),
            ),
            const SizedBox(width: 14),
              
            // Name, Author, Version
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  // Name
                  Text(
                    name,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                      color: hasData ? colorScheme.onSurface : colorScheme.onSurface.withOpacity(0.6),
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 5),
                  // Author & Version row
                  Row(
                    children: [
                      Icon(Icons.person_outline, size: 13, color: hasData ? colorScheme.secondary : colorScheme.outline),
                      const SizedBox(width: 4),
                      Flexible(
                        child: Text(
                          hasData && author.isNotEmpty ? author : 'Loading...', 
                          style: TextStyle(
                            fontSize: 12, 
                            color: hasData ? colorScheme.secondary : colorScheme.outline,
                            fontStyle: hasData ? FontStyle.normal : FontStyle.italic,
                          ), 
                          maxLines: 1, 
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 10),
                      _buildVersionBadge(context, hasData && version.isNotEmpty ? version : '--'),
                    ],
                  ),
                ],
              ),
            ),
            
            const SizedBox(width: 10),
              
            // CID badge + See more button (always show both)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Always show CID badge with copy
                _buildCidBadge(context),
                // Always show "See more" if callback provided
                if (onPullDetails != null) ...[
                  const SizedBox(width: 8),
                  OutlinedButton(
                    onPressed: onPullDetails,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: colorScheme.primary,
                      side: BorderSide(color: colorScheme.primary.withOpacity(0.5)),
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                      minimumSize: const Size(0, 32),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                    ),
                    child: Text('See more', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500)),
                  ),
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }
  
  Widget _buildSkillChip(BuildContext context, String name, String? id) {
    final colorScheme = Theme.of(context).colorScheme;
    final displayName = name.split('/').last.replaceAll('_', ' ');
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: colorScheme.tertiaryContainer.withOpacity(0.5),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: colorScheme.tertiary.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(displayName, style: TextStyle(fontSize: 11, color: colorScheme.onTertiaryContainer)),
          if (id != null) ...[
            const SizedBox(width: 4),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
              decoration: BoxDecoration(color: colorScheme.surface, borderRadius: BorderRadius.circular(3)),
              child: Text(id, style: TextStyle(fontSize: 9, fontFamily: 'monospace', color: colorScheme.onSurface.withOpacity(0.5))),
            ),
          ],
        ],
      ),
    );
  }

  String _formatDate(String isoDate) {
    try {
      final date = DateTime.parse(isoDate);
      return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
    } catch (_) {
      return isoDate;
    }
  }

  Widget _buildLoadButton(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return InkWell(
      onTap: onPullDetails,
      borderRadius: BorderRadius.circular(6),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: colorScheme.primary,
          borderRadius: BorderRadius.circular(6),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.download_rounded, size: 12, color: colorScheme.onPrimary),
            const SizedBox(width: 4),
            Text(
              'Load',
              style: TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w600,
                color: colorScheme.onPrimary,
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _extractDisplayName(String fullName) {
    // Extract just the agent name from full path like "org/repo/agent-name"
    final parts = fullName.split('/');
    return parts.isNotEmpty ? parts.last : fullName;
  }

  Widget _buildVersionBadge(BuildContext context, String version) {
    final colorScheme = Theme.of(context).colorScheme;
    
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: colorScheme.secondaryContainer.withOpacity(0.5),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        version,
        style: TextStyle(
          fontSize: 10,
          fontWeight: FontWeight.w600,
          color: colorScheme.onSecondaryContainer,
        ),
      ),
    );
  }

  Widget _buildCidBadge(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final shortCid = cid.length > 12
        ? '${cid.substring(0, 8)}...${cid.substring(cid.length - 4)}'
        : cid;

    return InkWell(
      onTap: () => _copyCid(context),
      borderRadius: BorderRadius.circular(6),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: colorScheme.primaryContainer.withOpacity(0.3),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(
            color: colorScheme.primary.withOpacity(0.2),
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.fingerprint_rounded,
              size: 12,
              color: colorScheme.primary,
            ),
            const SizedBox(width: 4),
            Text(
              shortCid,
              style: TextStyle(
                fontSize: 10,
                fontFamily: 'monospace',
                color: colorScheme.primary,
              ),
            ),
            const SizedBox(width: 4),
            Icon(
              Icons.copy_rounded,
              size: 11,
              color: colorScheme.primary,
            ),
          ],
        ),
      ),
    );
  }

  void _copyCid(BuildContext context) {
    Clipboard.setData(ClipboardData(text: cid));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: const Row(
          children: [
            Icon(Icons.check_circle, color: Colors.white, size: 18),
            SizedBox(width: 8),
            Text('CID copied to clipboard'),
          ],
        ),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      ),
    );
  }
}

/// Full detail card (expandable, starts expanded) - shown when "Load" is clicked
class AgentDetailCard extends StatefulWidget {
  final String cid;
  final Map<String, dynamic>? agentData;

  const AgentDetailCard({
    super.key,
    required this.cid,
    this.agentData,
  });

  @override
  State<AgentDetailCard> createState() => _AgentDetailCardState();
}

class _AgentDetailCardState extends State<AgentDetailCard> {
  bool _isExpanded = true; // Start expanded by default

  Map<String, dynamic>? _extractAgentInfo() {
    final data = widget.agentData;
    if (data == null) return null;
    if (data.containsKey('name') && data['name'] != null) {
      return data;
    } else if (data['data'] is Map) {
      return Map<String, dynamic>.from(data['data'] as Map);
    }
    return data;
  }

  String _extractDisplayName(String fullName) {
    final parts = fullName.split('/');
    return parts.isNotEmpty ? parts.last : fullName;
  }

  void _copyCid(BuildContext context) {
    Clipboard.setData(ClipboardData(text: widget.cid));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('CID copied!'), duration: Duration(seconds: 1)),
    );
  }

  String _formatDate(String isoDate) {
    try {
      final date = DateTime.parse(isoDate);
      return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
    } catch (_) {
      return isoDate;
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final agentInfo = _extractAgentInfo();
    
    final hasData = agentInfo != null && agentInfo.containsKey('name');
    
    final name = _extractDisplayName(agentInfo?['name']?.toString() ?? 'Unknown Agent');
    final authors = agentInfo?['authors'];
    final author = authors is List && authors.isNotEmpty ? authors.first.toString() : 'Unknown';
    final version = agentInfo?['version']?.toString() ?? '';
    final description = agentInfo?['description']?.toString() ?? '';
    final skills = agentInfo?['skills'] as List? ?? [];
    final domains = agentInfo?['domains'] as List? ?? [];
    final locators = agentInfo?['locators'] as List? ?? [];
    final modules = agentInfo?['modules'] as List? ?? [];
    final annotations = agentInfo?['annotations'] as Map<String, dynamic>? ?? {};
    final schemaVersion = agentInfo?['schema_version']?.toString() ?? '';
    final createdAt = agentInfo?['created_at']?.toString() ?? '';
    
    return Container(
      margin: const EdgeInsets.symmetric(vertical: 8),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.primary.withOpacity(0.4),
          width: 1.5,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.12),
            blurRadius: 12,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header (clickable to expand/collapse)
          InkWell(
            onTap: () => setState(() => _isExpanded = !_isExpanded),
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
            child: Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: colorScheme.primaryContainer.withOpacity(0.15),
                borderRadius: BorderRadius.vertical(
                  top: const Radius.circular(16),
                  bottom: _isExpanded ? Radius.zero : const Radius.circular(16),
                ),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Agent icon
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [
                          colorScheme.primary.withOpacity(0.3),
                          colorScheme.tertiary.withOpacity(0.3),
                        ],
                      ),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Icon(
                      Icons.smart_toy_rounded,
                      color: colorScheme.primary,
                      size: 26,
                    ),
                  ),
                  const SizedBox(width: 14),
                  
                  // Main info
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          name,
                          style: TextStyle(
                            fontSize: 17,
                            fontWeight: FontWeight.bold,
                            color: colorScheme.onSurface,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            Icon(Icons.person_outline, size: 14, color: colorScheme.secondary),
                            const SizedBox(width: 4),
                            Text(author, style: TextStyle(fontSize: 13, color: colorScheme.secondary)),
                            if (version.isNotEmpty) ...[
                              const SizedBox(width: 12),
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                                decoration: BoxDecoration(
                                  color: colorScheme.secondaryContainer.withOpacity(0.5),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Text(version, style: TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: colorScheme.onSecondaryContainer)),
                              ),
                            ],
                          ],
                        ),
                      ],
                    ),
                  ),
                  
                  // CID + expand icon
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      // CID badge
                      InkWell(
                        onTap: () => _copyCid(context),
                        borderRadius: BorderRadius.circular(6),
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                          decoration: BoxDecoration(
                            color: colorScheme.primaryContainer.withOpacity(0.3),
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(color: colorScheme.primary.withOpacity(0.2)),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(Icons.fingerprint_rounded, size: 12, color: colorScheme.primary),
                              const SizedBox(width: 4),
                              Text(
                                widget.cid.length > 12 
                                    ? '${widget.cid.substring(0, 8)}...${widget.cid.substring(widget.cid.length - 4)}'
                                    : widget.cid.isNotEmpty ? widget.cid : 'N/A',
                                style: TextStyle(fontSize: 10, fontFamily: 'monospace', color: colorScheme.primary),
                              ),
                              const SizedBox(width: 4),
                              Icon(Icons.copy_rounded, size: 11, color: colorScheme.primary),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 8),
                      Icon(
                        _isExpanded ? Icons.expand_less : Icons.expand_more,
                        color: colorScheme.primary,
                        size: 22,
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          
          // Expandable content
          if (_isExpanded) ...[
            // Description
            if (description.isNotEmpty)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  description,
                  style: TextStyle(
                    fontSize: 14,
                    color: colorScheme.onSurface.withOpacity(0.85),
                    height: 1.5,
                  ),
                ),
              ),
            
            // Skills
            if (skills.isNotEmpty)
              _buildDetailSection(
                context,
                icon: Icons.psychology_outlined,
                title: 'Skills (${skills.length})',
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: skills.map((s) {
                    final skillName = s is Map ? (s['name'] ?? s['class_name'] ?? 'skill') : s.toString();
                    final skillId = s is Map ? s['id']?.toString() : null;
                    return Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          colors: [colorScheme.tertiaryContainer.withOpacity(0.6), colorScheme.primaryContainer.withOpacity(0.4)],
                        ),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(skillName.toString().split('/').last.replaceAll('_', ' '), style: TextStyle(fontSize: 12, color: colorScheme.onTertiaryContainer)),
                          if (skillId != null) ...[
                            const SizedBox(width: 6),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                              decoration: BoxDecoration(color: colorScheme.surface.withOpacity(0.7), borderRadius: BorderRadius.circular(4)),
                              child: Text(skillId, style: TextStyle(fontSize: 9, fontFamily: 'monospace', color: colorScheme.primary)),
                            ),
                          ],
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            
            // Domains
            if (domains.isNotEmpty)
              _buildDetailSection(
                context,
                icon: Icons.category_outlined,
                title: 'Domains',
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: domains.map((d) {
                    final name = d is Map ? (d['name'] ?? 'domain') : d.toString();
                    return Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        color: colorScheme.secondaryContainer.withOpacity(0.5),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(name.toString().replaceAll('_', ' '), style: TextStyle(fontSize: 12, color: colorScheme.onSecondaryContainer)),
                    );
                  }).toList(),
                ),
              ),
            
            // Modules
            if (modules.isNotEmpty)
              _buildDetailSection(
                context,
                icon: Icons.extension_outlined,
                title: 'Modules (${modules.length})',
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: modules.map((m) {
                    final moduleName = m is Map ? (m['name'] ?? 'module') : m.toString();
                    final moduleId = m is Map ? m['id']?.toString() : null;
                    return Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        color: colorScheme.primaryContainer.withOpacity(0.4),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: colorScheme.primary.withOpacity(0.2)),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(moduleName.toString().split('/').last.replaceAll('_', ' '), style: TextStyle(fontSize: 12, color: colorScheme.onPrimaryContainer)),
                          if (moduleId != null) ...[
                            const SizedBox(width: 6),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                              decoration: BoxDecoration(color: colorScheme.surface, borderRadius: BorderRadius.circular(4)),
                              child: Text(moduleId, style: TextStyle(fontSize: 9, fontFamily: 'monospace', color: colorScheme.primary)),
                            ),
                          ],
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            
            // Annotations
            if (annotations.isNotEmpty)
              _buildDetailSection(
                context,
                icon: Icons.label_outlined,
                title: 'Annotations',
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: annotations.entries.map((e) {
                    return Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        color: colorScheme.secondaryContainer.withOpacity(0.3),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: colorScheme.secondary.withOpacity(0.2)),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text('${e.key}:', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: colorScheme.secondary)),
                          const SizedBox(width: 4),
                          Text(e.value.toString(), style: TextStyle(fontSize: 11, color: colorScheme.onSecondaryContainer)),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            
            // Locators
            if (locators.isNotEmpty)
              _buildDetailSection(
                context,
                icon: Icons.link_outlined,
                title: 'Locators',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: locators.map((l) {
                    final type = l is Map ? (l['type'] ?? 'url') : 'url';
                    final url = l is Map ? (l['url'] ?? '') : l.toString();
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: Row(
                        children: [
                          Icon(
                            type == 'docker_image' ? Icons.layers_outlined : Icons.language_outlined,
                            size: 14,
                            color: colorScheme.primary,
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              url.toString(),
                              style: TextStyle(fontSize: 12, color: colorScheme.primary, decoration: TextDecoration.underline),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            
            // Metadata footer
            if (schemaVersion.isNotEmpty || createdAt.isNotEmpty)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                child: Row(
                  children: [
                    if (schemaVersion.isNotEmpty) ...[
                      Icon(Icons.schema_outlined, size: 12, color: colorScheme.outline),
                      const SizedBox(width: 4),
                      Text('Schema $schemaVersion', style: TextStyle(fontSize: 10, color: colorScheme.outline)),
                    ],
                    if (schemaVersion.isNotEmpty && createdAt.isNotEmpty)
                      const SizedBox(width: 16),
                    if (createdAt.isNotEmpty) ...[
                      Icon(Icons.calendar_today_outlined, size: 12, color: colorScheme.outline),
                      const SizedBox(width: 4),
                      Text(_formatDate(createdAt), style: TextStyle(fontSize: 10, color: colorScheme.outline)),
                    ],
                  ],
                ),
              ),
          ],
        ],
      ),
    );
  }

  Widget _buildDetailSection(BuildContext context, {required IconData icon, required String title, required Widget child}) {
    final colorScheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 16, color: colorScheme.primary),
              const SizedBox(width: 6),
              Text(title, style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: colorScheme.onSurface)),
            ],
          ),
          const SizedBox(height: 10),
          child,
        ],
      ),
    );
  }
}
