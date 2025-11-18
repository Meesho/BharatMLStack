package external

import (
	"strings"
)

func FlattenResponseMap(isResponse *ISResponse, userId string) map[string]map[string]string {

	interactionMap := make(map[string]map[string]string)
	for _, interactionType := range isResponse.InteractionTypeToInteractions {
		interactions := interactionType.Interactions
		if len(interactions) > 0 {
			entityType := interactions[0].Type
			key := strings.ToLower(entityType + ":" + interactionType.InteractionType)
			ids := make([]string, 0)

			for _, interaction := range interactions {
				ids = append(ids, interaction.Id)
			}
			interactionMap[key] = map[string]string{userId: strings.Join(ids, ",")}
		}
	}
	return interactionMap
}
