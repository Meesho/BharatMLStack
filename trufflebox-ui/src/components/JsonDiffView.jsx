import React, { useMemo } from 'react';
import { Diff, Hunk, parseDiff } from 'react-diff-view';
import { Box, Typography, Paper, Alert, Chip } from '@mui/material';
import 'react-diff-view/style/index.css';

const JsonDiffView = ({ 
  originalText = '', 
  changedText = '', 
  title = 'React Diff View Test'
}) => {
  
  // Create a proper unified diff format from the two texts
  const createUnifiedDiff = (original, changed) => {
    const originalLines = original.split('\n');
    const changedLines = changed.split('\n');
    
    // Create proper unified diff header
    let diffText = '--- a/original.json\n+++ b/changed.json\n';
    
    // Calculate line ranges
    const originalLineCount = originalLines.length;
    const changedLineCount = changedLines.length;
    
    // Create hunk header
    diffText += `@@ -1,${originalLineCount} +1,${changedLineCount} @@\n`;
    
    // Use a more sophisticated diff algorithm
    const diff = computeLineDiff(originalLines, changedLines);
    
    // Convert diff to unified format
    for (const change of diff) {
      if (change.type === 'equal') {
        // Unchanged lines
        for (let i = 0; i < change.count; i++) {
          diffText += ' ' + (originalLines[change.start + i] || '') + '\n';
        }
      } else if (change.type === 'delete') {
        // Removed lines
        for (let i = 0; i < change.count; i++) {
          diffText += '-' + (originalLines[change.start + i] || '') + '\n';
        }
      } else if (change.type === 'insert') {
        // Added lines
        for (let i = 0; i < change.count; i++) {
          diffText += '+' + (changedLines[change.newStart + i] || '') + '\n';
        }
      }
    }
    
    return diffText;
  };

  // Simple diff algorithm for line-by-line comparison
  const computeLineDiff = (original, changed) => {
    const changes = [];
    let i = 0, j = 0;
    
    while (i < original.length || j < changed.length) {
      if (i >= original.length) {
        // Only changed lines left
        changes.push({ type: 'insert', start: i, newStart: j, count: changed.length - j });
        break;
      } else if (j >= changed.length) {
        // Only original lines left
        changes.push({ type: 'delete', start: i, newStart: j, count: original.length - i });
        break;
      } else if (original[i] === changed[j]) {
        // Lines are equal
        let equalCount = 0;
        while (i + equalCount < original.length && 
               j + equalCount < changed.length && 
               original[i + equalCount] === changed[j + equalCount]) {
          equalCount++;
        }
        changes.push({ type: 'equal', start: i, newStart: j, count: equalCount });
        i += equalCount;
        j += equalCount;
      } else {
        // Lines are different - find the best match
        let deleteCount = 0;
        let insertCount = 0;
        
        // Look ahead to find the next match
        let foundMatch = false;
        const maxLookAhead = Math.min(10, original.length - i, changed.length - j);
        
        for (let k = 1; k <= maxLookAhead; k++) {
          if (i + k < original.length && original[i + k] === changed[j]) {
            deleteCount = k;
            foundMatch = true;
            break;
          }
          if (j + k < changed.length && original[i] === changed[j + k]) {
            insertCount = k;
            foundMatch = true;
            break;
          }
        }
        
        if (!foundMatch) {
          // No match found, treat as single line change
          deleteCount = 1;
          insertCount = 1;
        }
        
        if (deleteCount > 0) {
          changes.push({ type: 'delete', start: i, newStart: j, count: deleteCount });
          i += deleteCount;
        }
        if (insertCount > 0) {
          changes.push({ type: 'insert', start: i, newStart: j, count: insertCount });
          j += insertCount;
        }
      }
    }
    
    return changes;
  };

  // Parse the diff using useMemo for performance
  const files = useMemo(() => {
    try {
      const diffText = createUnifiedDiff(originalText, changedText);
      return parseDiff(diffText);
    } catch (error) {
      console.error('Error parsing diff:', error);
      return [];
    }
  }, [originalText, changedText]);

  // Calculate statistics
  const stats = useMemo(() => {
    const originalLines = originalText.split('\n').length;
    const changedLines = changedText.split('\n').length;
    let additions = 0;
    let deletions = 0;

    // Count additions and deletions from the parsed files
    files.forEach(file => {
      file.hunks.forEach(hunk => {
        hunk.changes.forEach(change => {
          if (change.type === 'insert') additions++;
          if (change.type === 'delete') deletions++;
        });
      });
    });

    return { originalLines, changedLines, additions, deletions };
  }, [files, originalText, changedText]);

  return (
    <Box sx={{ 
      width: '100%', 
      height: '100%',
      backgroundColor: '#ffffff',
      border: '1px solid #e1e4e8',
      borderRadius: '6px',
      overflow: 'hidden'
    }}>
      {/* Header */}
      <Box sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '12px 16px',
        backgroundColor: '#f6f8fa',
        borderBottom: '1px solid #e1e4e8',
        minHeight: '48px'
      }}>
        <Typography variant="h6" sx={{ 
          fontSize: '16px',
          fontWeight: 600,
          color: '#24292f'
        }}>
          {title}
        </Typography>
        
        {/* Statistics */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Chip 
            label={`+${stats.additions}`} 
            size="small" 
            sx={{ 
              backgroundColor: '#dafbe1', 
              color: '#1a7f37',
              fontSize: '11px',
              fontWeight: 600,
              height: '20px',
              border: '1px solid #a2f0ba',
              '& .MuiChip-label': { px: 1 }
            }} 
          />
          <Chip 
            label={`-${stats.deletions}`} 
            size="small" 
            sx={{ 
              backgroundColor: '#ffeef0', 
              color: '#cf222e',
              fontSize: '11px',
              fontWeight: 600,
              height: '20px',
              border: '1px solid #ffbdbd',
              '& .MuiChip-label': { px: 1 }
            }} 
          />
          <Typography variant="caption" sx={{ 
            color: '#656d76',
            fontSize: '11px',
            fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif',
            backgroundColor: '#ffffff',
            padding: '2px 6px',
            borderRadius: '4px',
            border: '1px solid #d0d7de'
          }}>
            {stats.originalLines} → {stats.changedLines}
          </Typography>
        </Box>
      </Box>

      {/* Diff Content */}
      <Box sx={{ 
        height: '100%',
        overflow: 'auto'
      }}>
        {files.length === 0 ? (
          <Alert severity="info" sx={{ margin: '16px' }}>
            No differences found or error parsing diff.
          </Alert>
        ) : (
          files.map(({ hunks, oldRevision, newRevision, type, ...props }, index) => (
            <Box key={`${oldRevision}-${newRevision}-${index}`} sx={{ marginBottom: '16px' }}>
              <Paper elevation={1} sx={{ 
                padding: '16px',
                '& .diff-view': {
                  fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                  fontSize: '12px',
                  lineHeight: '20px'
                }
              }}>
                
                <Diff
                  viewType="split"
                  diffType={type}
                  hunks={hunks}
                  {...props}
                >
                  {hunks => hunks.map(hunk => (
                    <Hunk key={hunk.content} hunk={hunk} />
                  ))}
                </Diff>
              </Paper>
            </Box>
          ))
        )}
      </Box>
    </Box>
  );
};

export default JsonDiffView;
