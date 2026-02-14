package handler

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
)

func parseGCSURL(gcsURL string) (bucket, objectPath string, ok bool) {
	if strings.HasPrefix(gcsURL, "gcs://") {
		gcsURL = strings.Replace(gcsURL, "gcs://", "gs://", 1)
	}

	if !strings.HasPrefix(gcsURL, gcsPrefix) {
		return constant.EmptyString, constant.EmptyString, false
	}

	trimmed := strings.TrimPrefix(gcsURL, gcsPrefix)
	parts := strings.SplitN(trimmed, slashConstant, 2)
	if len(parts) < 1 {
		return constant.EmptyString, constant.EmptyString, false
	}

	bucket = parts[0]
	if len(parts) == 2 {
		objectPath = parts[1]
	}
	return bucket, objectPath, true
}

func extractGCSPath(gcsURL string) (bucket, objectPath string) {
	bucket, objectPath, ok := parseGCSURL(gcsURL)
	if !ok {
		return constant.EmptyString, constant.EmptyString
	}
	return bucket, objectPath
}

func extractGCSDetails(gcsURL string) (bucket, pathOnly, modelName string) {
	bucket, objectPath, ok := parseGCSURL(gcsURL)
	if !ok || objectPath == constant.EmptyString {
		return constant.EmptyString, constant.EmptyString, constant.EmptyString
	}

	segments := strings.Split(objectPath, slashConstant)
	if len(segments) == 0 {
		return bucket, constant.EmptyString, constant.EmptyString
	}

	modelName = segments[len(segments)-1]
	pathOnly = path.Join(segments[:len(segments)-1]...)
	return bucket, pathOnly, modelName
}

// GetDerivedModelName returns the derived model name with deployable tag
func (p *Predator) GetDerivedModelName(payloadObject Payload, requestType string) (string, error) {
	if requestType != ScaleUpRequestType {
		return payloadObject.ModelName, nil
	}
	serviceDeployableID := payloadObject.ConfigMapping.ServiceDeployableID
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(int(serviceDeployableID))
	if err != nil {
		return constant.EmptyString, fmt.Errorf("%s: %w", errFetchDeployableConfig, err)
	}

	deployableTag := serviceDeployable.DeployableTag
	if deployableTag == "" {
		return payloadObject.ModelName, nil
	}

	derivedModelName := payloadObject.ModelName + deployableTagDelimiter + deployableTag
	derivedModelName = derivedModelName + deployableTagDelimiter + scaleupTag
	return derivedModelName, nil
}

// GetOriginalModelName returns the original model name if no tag is found (backward compatibility)
func (p *Predator) GetOriginalModelName(derivedModelName string, serviceDeployableID int) (string, error) {
	serviceDeployable, err := p.ServiceDeployableRepo.GetById(serviceDeployableID)
	if err != nil {
		return constant.EmptyString, fmt.Errorf("%s: %w", errFetchDeployableConfig, err)
	}

	deployableTag := serviceDeployable.DeployableTag
	if deployableTag == "" {
		return derivedModelName, nil
	}

	scaleupSuffix := deployableTagDelimiter + scaleupTag
	derivedModelName = strings.TrimSuffix(derivedModelName, scaleupSuffix)

	deployableTagSuffix := deployableTagDelimiter + deployableTag
	if originalName, foundSuffix := strings.CutSuffix(derivedModelName, deployableTagSuffix); foundSuffix {
		return originalName, nil
	}

	return derivedModelName, nil
}

func (p *Predator) isNonProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(pred.AppEnv))
	if env == "prd" || env == "prod" {
		return false
	}
	return true
}

func replaceInstanceCountInConfigPreservingFormat(data []byte, newCount int) ([]byte, error) {
	input := string(data)
	var output strings.Builder

	inInstanceGroup := false
	bracket := 0
	brace := 0

	i := 0
	for i < len(input) {

		if !inInstanceGroup && strings.HasPrefix(input[i:], "instance_group") {
			inInstanceGroup = true
			output.WriteString("instance_group")
			i += len("instance_group")
			continue
		}

		ch := input[i]

		if inInstanceGroup {
			if ch == '[' {
				bracket++
			}
			if ch == ']' {
				bracket--
				if bracket == 0 {
					inInstanceGroup = false
				}
			}

			if ch == '{' {
				brace++
			}
			if ch == '}' {
				brace--
			}
			if brace > 0 && strings.HasPrefix(input[i:], "count") {
				j := i + len("count")
				for j < len(input) && unicode.IsSpace(rune(input[j])) {
					j++
				}
				if j < len(input) && input[j] == ':' {
					j++
					for j < len(input) && unicode.IsSpace(rune(input[j])) {
						j++
					}
					k := j
					for k < len(input) && unicode.IsDigit(rune(input[k])) {
						k++
					}
					output.WriteString(input[i:j])
					output.WriteString(strconv.Itoa(newCount))
					output.WriteString(input[k:])
					return []byte(output.String()), nil
				}
			}
		}
		output.WriteByte(ch)
		i++
	}
	return nil, fmt.Errorf("%s", errNoInstanceGroup)
}
