import { useCallback } from 'react';

/**
 * Custom hook that provides date formatting utilities
 * @returns {Object} Date formatting functions
 */
const useFormatDate = () => {
  /**
   * Converts UTC date string to IST format
   * @param {string} utcDateString - Date string in UTC format
   * @returns {string} Formatted date string in IST
   */
  const formatDateToIST = useCallback((utcDateString) => {
    if (!utcDateString) return "";
    const date = new Date(utcDateString);
    return date.toLocaleString('en-IN', { 
      timeZone: 'Asia/Kolkata',
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true
    });
  }, []);

  return {
    formatDateToIST
  };
};

export default useFormatDate;