/**
 * Shared Components Library for Embedding Platform
 * 
 * This module exports all reusable components and hooks
 * to reduce code duplication across the platform.
 */

// Components
export { default as StatusChip } from './StatusChip';
export { default as StatusFilterHeader } from './StatusFilterHeader';
export { default as ViewDetailModal } from './ViewDetailModal';

// Hooks
export { default as useTableFilter, useTableFilter as useTableFilterNamed } from './hooks/useTableFilter';
export { default as useStatusFilter, useStatusFilter as useStatusFilterNamed } from './hooks/useStatusFilter';
export { default as useNotification } from './hooks/useNotification';
