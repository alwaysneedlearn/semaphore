package db

import (
	"strings"

	"github.com/semaphoreui/semaphore/util"
)

func ConvertFlatToNested(flatMap map[string]string) map[string]any {
	nestedMap := make(map[string]any)

	for key, value := range flatMap {
		parts := strings.Split(key, ".")
		currentMap := nestedMap

		for i, part := range parts {
			if i == len(parts)-1 {
				currentMap[part] = value
			} else {
				if _, exists := currentMap[part]; !exists {
					currentMap[part] = make(map[string]any)
				}
				currentMap = currentMap[part].(map[string]any)
			}
		}
	}

	return nestedMap
}

// optionKeysExcludedFromConfigMerge are stored as opaque JSON/strings and must not
// be split on "." into util.Config (e.g. tdengine.config → tdengine.config panic).
var optionKeysExcludedFromConfigMerge = map[string]bool{
	"tdengine.config":  true,
	"tdengine_config":  true,
}

func filterOptionsForConfigMerge(opts map[string]string) map[string]string {
	if len(opts) == 0 {
		return opts
	}
	out := make(map[string]string, len(opts))
	for k, v := range opts {
		if optionKeysExcludedFromConfigMerge[k] {
			continue
		}
		out[k] = v
	}
	return out
}

func FillConfigFromDB(store Store) (err error) {

	opts, err := store.GetOptions(RetrieveQueryParams{})

	if err != nil {
		return
	}

	options := ConvertFlatToNested(filterOptionsForConfigMerge(opts))

	if options["apps"] == nil {
		options["apps"] = make(map[string]any)
	}

	err = util.AssignMapToStruct(options, util.Config)

	return
}
