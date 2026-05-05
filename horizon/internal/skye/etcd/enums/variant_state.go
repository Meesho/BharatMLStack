package enums

// VariantState represents the various states that a variant can be in during processing.
type VariantState string

// Constants representing different variant states.
const (
	DATA_INGESTION_STARTED        VariantState = "DATA_INGESTION_STARTED"        // Data ingestion has started.
	DATA_INGESTION_COMPLETED      VariantState = "DATA_INGESTION_COMPLETED"      // Data ingestion has completed successfully.
	INDEXING_STARTED              VariantState = "INDEXING_STARTED"              // Indexing process has started.
	INDEXING_IN_PROGRESS          VariantState = "INDEXING_IN_PROGRESS"          // Indexing is currently in progress.
	INDEXING_COMPLETED_WITH_RESET VariantState = "INDEXING_COMPLETED_WITH_RESET" // Indexing completed, and the state has been reset.
	INDEXING_COMPLETED            VariantState = "INDEXING_COMPLETED"            // Indexing completed successfully.
	MODEL_VERSION_UPDATED         VariantState = "MODEL_VERSION_UPDATED"         // Model version has been updated.
	COMPLETED                     VariantState = "COMPLETED"                     // The variant processing has been completed.
)
