import { useState, useCallback } from 'react';

/**
 * Shared hook for Snackbar-based notifications across Embedding Platform.
 * @returns {{ notification: { open, message, severity }, showNotification: (message, severity) => void, closeNotification: () => void }}
 */
export const useNotification = (initialState = { open: false, message: '', severity: 'success' }) => {
  const [notification, setNotification] = useState(initialState);

  const showNotification = useCallback((message, severity = 'success') => {
    setNotification({ open: true, message, severity });
  }, []);

  const closeNotification = useCallback(() => {
    setNotification((prev) => ({ ...prev, open: false }));
  }, []);

  return { notification, showNotification, closeNotification };
};

export default useNotification;
