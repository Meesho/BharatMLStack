// Database types supported for variant onboarding
export const DATABASE_TYPES = {
  QDRANT: 'QDRANT',
  // ELASTICSEARCH: 'ELASTICSEARCH',
  // PINECONE: 'PINECONE',
  // WEAVIATE: 'WEAVIATE',
};

// Database type options for dropdown
export const DATABASE_TYPE_OPTIONS = [
  {
    value: DATABASE_TYPES.QDRANT,
    label: 'Qdrant',
    description: 'Vector database for similarity search and machine learning applications'
  }
];

// Default database type
export const DEFAULT_DATABASE_TYPE = DATABASE_TYPES.QDRANT;
