import { useMemo } from 'react';

/**
 * Custom hook for filtering and sorting table data
 * 
 * @param {array} data - The data array to filter
 * @param {string} searchQuery - Search query string
 * @param {array} selectedStatuses - Array of selected status values to filter by
 * @param {function} searchFields - Function that returns array of searchable field values for an item
 * @param {string} sortField - Field name to sort by (defaults to 'created_at')
 * @param {string} sortOrder - 'asc' or 'desc' (defaults to 'desc')
 * @returns {array} Filtered and sorted data
 */
export const useTableFilter = ({
  data = [],
  searchQuery = '',
  selectedStatuses = [],
  searchFields = (item) => [],
  sortField = 'created_at',
  sortOrder = 'desc'
}) => {
  return useMemo(() => {
    let filtered = [...data].filter(item => item != null); // Filter out null/undefined items
    
    // Status filtering
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(item => {
        const status = item?.status;
        if (!status) return false;
        return selectedStatuses.includes(status.toUpperCase());
      });
    }
    
    // Search filtering
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      filtered = filtered.filter(item => {
        if (!item) return false;
        
        // Get searchable fields from the item
        const searchableFields = searchFields(item);
        
        // Check if any searchable field contains the query
        return searchableFields.some(field => 
          String(field || '').toLowerCase().includes(searchLower)
        );
      });
    }
    
    // Sorting
    filtered.sort((a, b) => {
      const aValue = a?.[sortField];
      const bValue = b?.[sortField];
      
      // Handle date sorting
      if (sortField === 'created_at' || sortField === 'updated_at') {
        const dateA = aValue ? new Date(aValue) : new Date(0);
        const dateB = bValue ? new Date(bValue) : new Date(0);
        return sortOrder === 'desc' ? dateB - dateA : dateA - dateB;
      }
      
      // Handle string/number sorting
      if (aValue < bValue) return sortOrder === 'desc' ? 1 : -1;
      if (aValue > bValue) return sortOrder === 'desc' ? -1 : 1;
      return 0;
    });
    
    return filtered;
  }, [data, searchQuery, selectedStatuses, searchFields, sortField, sortOrder]);
};

export default useTableFilter;
