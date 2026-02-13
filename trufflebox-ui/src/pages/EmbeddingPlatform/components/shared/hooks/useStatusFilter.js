import { useState, useCallback } from 'react';
import { REQUEST_STATUS } from '../../../../../services/embeddingPlatform/constants';

/**
 * Custom hook for managing status filter state
 * 
 * @param {array} initialStatuses - Initial selected statuses (defaults to all statuses)
 * @returns {object} Status filter state and handlers
 */
export const useStatusFilter = (initialStatuses = null) => {
  const defaultStatuses = initialStatuses || [
    REQUEST_STATUS.PENDING,
    REQUEST_STATUS.APPROVED,
    REQUEST_STATUS.REJECTED
  ];
  
  const [selectedStatuses, setSelectedStatuses] = useState(defaultStatuses);

  const handleStatusChange = useCallback((newStatuses) => {
    setSelectedStatuses(newStatuses);
  }, []);

  const resetStatusFilter = useCallback(() => {
    setSelectedStatuses(defaultStatuses);
  }, [defaultStatuses]);

  return {
    selectedStatuses,
    setSelectedStatuses,
    handleStatusChange,
    resetStatusFilter
  };
};

export default useStatusFilter;
