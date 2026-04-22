package pack

import (
	"encoding/json"
	"io/ioutil"

	"github.com/openmigrate/openmigrate/internal/core/types"
)

func WriteMeta(path string, meta types.PackageMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0o644)
}

func ReadMeta(path string) (types.PackageMeta, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return types.PackageMeta{}, err
	}
	var meta types.PackageMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return types.PackageMeta{}, err
	}
	if meta.SchemaVersion < 0 || meta.SchemaVersion > 1 {
		return types.PackageMeta{}, types.ErrSchemaVersion
	}
	return meta, nil
}
