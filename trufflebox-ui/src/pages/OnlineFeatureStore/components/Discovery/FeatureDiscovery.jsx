import React, { useState } from 'react';
import EntityDiscovery from './EntityDiscovery';
import FeatureGroupDiscovery from './FeatureGroupDiscovery';
import FeatureList from './FeatureList';

function FeatureDiscovery () {

    const [selectedEntity, setSelectedEntity] = useState(null);
    const [selectedFeatureGroup, setSelectedFeatureGroup] = useState(null);

    const handleEntityClick = (entity) => {
        setSelectedEntity(entity);
        setSelectedFeatureGroup(null); // Clear feature group when changing entity
    }

    const handleFeatureGroupClick = (featureGroup) => {
        setSelectedFeatureGroup(featureGroup);
    }

    return (
        <div style={{ display: 'flex', position: 'relative', gap: '20px' }}>
            <EntityDiscovery onEntityClick={handleEntityClick} />
            {selectedEntity && (
                <FeatureGroupDiscovery 
                    entityId={selectedEntity} 
                    onFeatureGroupClick={handleFeatureGroupClick} 
                />
            )}
            {selectedFeatureGroup && (
                <FeatureList 
                    features={selectedFeatureGroup.features} 
                    activeVersion={selectedFeatureGroup['active-version']} 
                    entityLabel={selectedFeatureGroup['entity-label']} 
                    featureGroupLabel={selectedFeatureGroup['feature-group-label']}
                />
            )}
        </div>
    );
}

export default FeatureDiscovery;
