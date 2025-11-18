import { useState } from 'react';
import ProductionCredentialModal from './ProductionCredentialModal';

/**
 * Custom hook for handling promotion with production credential verification
 */
export const usePromoteWithProdCredentials = () => {
  const [showProdCredentialModal, setShowProdCredentialModal] = useState(false);
  const [selectedItemForPromotion, setSelectedItemForPromotion] = useState(null);
  const [onPromoteCallback, setOnPromoteCallback] = useState(null);

  const initiatePromotion = (item, promoteCallback) => {
    setSelectedItemForPromotion(item);
    setOnPromoteCallback(() => promoteCallback);
    setShowProdCredentialModal(true);
  };

  const handleProductionCredentialSuccess = async (prodCredentials) => {
    setShowProdCredentialModal(false);
    
    if (selectedItemForPromotion && onPromoteCallback) {
      try {
        await onPromoteCallback(selectedItemForPromotion, prodCredentials);
      } catch (error) {
        console.error('Promotion failed:', error);
      } finally {
        setSelectedItemForPromotion(null);
        setOnPromoteCallback(null);
      }
    }
  };

  const handleProductionCredentialClose = () => {
    setShowProdCredentialModal(false);
    setSelectedItemForPromotion(null);
    setOnPromoteCallback(null);
  };

  const ProductionCredentialModalComponent = ({ title, description }) => (
    <ProductionCredentialModal
      open={showProdCredentialModal}
      onClose={handleProductionCredentialClose}
      onSuccess={handleProductionCredentialSuccess}
      title={title || "Production Credential Verification"}
      description={description || "Please enter your production credentials to proceed with promotion."}
    />
  );

  return {
    initiatePromotion,
    ProductionCredentialModalComponent,
    isShowingModal: showProdCredentialModal,
    selectedItem: selectedItemForPromotion
  };
};

/**
 * Example usage in a registry component:
 * 
 * const MyRegistryComponent = () => {
 *   const { initiatePromotion, ProductionCredentialModalComponent } = usePromoteWithProdCredentials();
 *   
 *   const handlePromote = (row) => {
 *     initiatePromotion(row, async (item, prodCredentials) => {
 *       // This is where you make the actual promotion API call
 *       const response = await fetch(`${URL_CONSTANTS.REACT_APP_HORIZON_PROD_BASE_URL}/api/v1/promote`, {
 *         method: 'POST',
 *         headers: {
 *           'Authorization': `Bearer ${prodCredentials.token}`,
 *           'Content-Type': 'application/json'
 *         },
 *         body: JSON.stringify({
 *           item_id: item.id,
 *           updated_by: prodCredentials.email
 *         })
 *       });
 *       
 *       if (!response.ok) {
 *         throw new Error('Promotion failed');
 *       }
 *       
 *       // Handle success (e.g., show notification, refresh data)
 *       console.log('Promotion successful');
 *     });
 *   };
 *   
 *   return (
 *     <div>
 *       <button onClick={() => handlePromote(someItem)}>Promote</button>
 *       <ProductionCredentialModalComponent 
 *         title="Promote Configuration"
 *         description="Enter production credentials to promote this configuration."
 *       />
 *     </div>
 *   );
 * };
 */

export default usePromoteWithProdCredentials; 