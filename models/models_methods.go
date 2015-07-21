package models

import (
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
)

// -------------------
// --- Struct methods

// Validate ...
func (env EnvironmentItemModel) Validate() error {
	key, _, err := env.GetKeyValuePair()
	if err != nil {
		return err
	}
	if key == "" {
		return errors.New("Invalid environment: empty env_key")
	}

	options, err := env.GetOptions()
	if err != nil {
		return err
	}

	if options.Title == nil || *options.Title == "" {
		return errors.New("Invalid environment: missing or empty title")
	}

	return nil
}

// Validate ...
func (step StepModel) Validate() error {
	if step.Title == nil || *step.Title == "" {
		return errors.New("Invalid step: missing or empty title")
	}
	if step.Summary == nil || *step.Summary == "" {
		return errors.New("Invalid step: missing or empty summary")
	}
	if step.Website == nil || *step.Website == "" {
		return errors.New("Invalid step: missing or empty website")
	}
	if step.Source.Git == nil || *step.Source.Git == "" {
		return errors.New("Invalid step: missing or empty source")
	}
	for _, input := range step.Inputs {
		err := input.Validate()
		if err != nil {
			return err
		}
	}
	for _, output := range step.Outputs {
		err := output.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetStep ...
func (collection StepCollectionModel) GetStep(id, version string) (StepModel, bool) {
	log.Debugln("-> GetStep")
	stepHash := collection.Steps
	//map[string]StepModel
	stepVersions, found := stepHash[id]
	if !found {
		return StepModel{}, false
	}
	step, found := stepVersions.Versions[version]
	if !found {
		return StepModel{}, false
	}
	return step, true
}

// GetDownloadLocations ...
func (collection StepCollectionModel) GetDownloadLocations(id, version string) ([]DownloadLocationModel, error) {
	locations := []DownloadLocationModel{}
	for _, downloadLocation := range collection.DownloadLocations {
		switch downloadLocation.Type {
		case "zip":
			url := downloadLocation.Src + id + "/" + version + "/step.zip"
			location := DownloadLocationModel{
				Type: downloadLocation.Type,
				Src:  url,
			}
			locations = append(locations, location)
		case "git":
			step, found := collection.GetStep(id, version)
			if found {
				location := DownloadLocationModel{
					Type: downloadLocation.Type,
					Src:  *step.Source.Git,
				}
				locations = append(locations, location)
			}
		default:
			return []DownloadLocationModel{}, fmt.Errorf("[STEPMAN] - Invalid download location (%#v) for step (%#v)", downloadLocation, id)
		}
	}
	if len(locations) < 1 {
		return []DownloadLocationModel{}, fmt.Errorf("[STEPMAN] - No download location found for step (%#v)", id)
	}
	return locations, nil
}

// GetKeyValuePair ...
func (env EnvironmentItemModel) GetKeyValuePair() (string, string, error) {
	if len(env) < 3 {
		retKey := ""
		retValue := ""

		for key, value := range env {
			if key != optionsKey {
				if retKey != "" {
					return "", "", errors.New("Invalid env: more then 1 key-value field found!")
				}

				valueStr, ok := value.(string)
				if ok == false {
					return "", "", fmt.Errorf("Invalid value (key:%#v) (value:%#v)", key, value)
				}

				retKey = key
				retValue = valueStr
			}
		}

		if retKey == "" {
			return "", "", errors.New("Invalid env: no envKey specified!")
		}

		return retKey, retValue, nil
	}

	return "", "", errors.New("Invalid env: more then 2 fields ")
}

// ParseFromInterfaceMap ...
func (envSerModel *EnvironmentItemOptionsModel) ParseFromInterfaceMap(input map[interface{}]interface{}) error {
	for key, value := range input {
		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("Invalid key, should be a string: %#v", key)
		}
		switch keyStr {
		case "title":
			castedValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
			}
			*envSerModel.Title = castedValue
		case "description":
			castedValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
			}
			*envSerModel.Description = castedValue
		case "value_options":
			castedValue, ok := value.([]string)
			if !ok {
				// try with []interface{} instead and cast the
				//  items to string
				castedValue = []string{}
				interfArr, ok := value.([]interface{})
				if !ok {
					return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
				}
				for _, interfItm := range interfArr {
					castedItm, ok := interfItm.(string)
					if !ok {
						return fmt.Errorf("Invalid value in value_options (%#v), not a string: %#v", interfArr, interfItm)
					}
					castedValue = append(castedValue, castedItm)
				}
			}
			envSerModel.ValueOptions = castedValue
		case "is_required":
			castedValue, ok := value.(bool)
			if !ok {
				return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
			}
			envSerModel.IsRequired = &castedValue
		case "is_expand":
			castedValue, ok := value.(bool)
			if !ok {
				return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
			}
			envSerModel.IsExpand = &castedValue
		case "is_dont_change_value":
			castedValue, ok := value.(bool)
			if !ok {
				return fmt.Errorf("Invalid value type (key:%s): %#v", keyStr, value)
			}
			envSerModel.IsDontChangeValue = &castedValue
		default:
			return fmt.Errorf("Not supported key found in options: %#v", key)
		}
	}
	return nil
}

// GetOptions ...
func (env EnvironmentItemModel) GetOptions() (EnvironmentItemOptionsModel, error) {
	if len(env) > 2 {
		return EnvironmentItemOptionsModel{}, errors.New("Invalid env: more then 2 field")
	}

	optsShouldExist := false
	if len(env) == 2 {
		optsShouldExist = true
	}

	value, found := env[optionsKey]
	if !found {
		if optsShouldExist {
			return EnvironmentItemOptionsModel{}, errors.New("Invalid env: 2 fields but, no opts found")
		}
		return EnvironmentItemOptionsModel{}, nil
	}

	envItmCasted, ok := value.(EnvironmentItemOptionsModel)
	if ok {
		return envItmCasted, nil
	}

	// if it's read from a file (YAML/JSON) then it's most likely not the proper type
	//  so cast it from the generic interface-interface map
	optionsInterfaceMap, ok := value.(map[interface{}]interface{})
	if !ok {
		return EnvironmentItemOptionsModel{}, fmt.Errorf("Invalid options (value:%#v) - failed to map-interface cast", value)
	}

	options := EnvironmentItemOptionsModel{}
	err := options.ParseFromInterfaceMap(optionsInterfaceMap)
	if err != nil {
		return EnvironmentItemOptionsModel{}, err
	}

	log.Debugf("Parsed options: %#v\n", options)

	return options, nil
}
