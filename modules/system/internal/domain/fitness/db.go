package fitness

import (
	"context"

	"platform/ent"
	"platform/ent/sysstoragefile"
	"platform/ent/sysuser"
	"platform/modules/shared"
)

func loadFiles(ctx context.Context, ids []string) (map[string]*ent.SysStorageFile, error) {
	clean := uniqueNonEmpty(ids)
	result := make(map[string]*ent.SysStorageFile, len(clean))
	if len(clean) == 0 {
		return result, nil
	}
	items, err := shared.EntClient.SysStorageFile.Query().Where(sysstoragefile.IDIn(clean...)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

func loadUsers(ctx context.Context, ids []string) (map[string]*ent.SysUser, error) {
	clean := uniqueNonEmpty(ids)
	result := make(map[string]*ent.SysUser, len(clean))
	if len(clean) == 0 {
		return result, nil
	}
	items, err := shared.EntClient.SysUser.Query().Where(sysuser.IDIn(clean...)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.ID] = item
	}
	return result, nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
