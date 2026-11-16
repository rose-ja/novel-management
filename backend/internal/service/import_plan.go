package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"novel-management/backend/internal/imports"
	"novel-management/backend/internal/model"
	"novel-management/backend/internal/repository"
	"novel-management/backend/internal/search"
)

// importPlan 是一次原子提交所需的全部行，按外键顺序持有。
type importPlan struct {
	work             model.Work
	categories       []model.SystemCategory
	nodes            []model.SystemNode
	factions         []model.Faction
	locations        []model.Location
	characters       []model.Character
	names            []model.CharacterName
	refs             []model.CharacterSystemRef
	charLocations    []model.CharacterLocation
	locationFactions []model.LocationFaction
	rankings         []model.Ranking
	seats            []model.RankingSeat
	chronicles       []model.Chronicle
	media            []model.MediaAsset
	counts           model.ImportCounts
	images           model.ImportImageSummary
	warnings         []model.ImportIssue
}

func buildImportPlan(workID, name, nameKey string, spec *imports.Spec, media map[string]model.MediaAsset, now time.Time) *importPlan {
	plan := &importPlan{
		work: model.Work{
			ID: workID, Name: name, NameKey: nameKey,
			Description: trimmed(spec.Work.Description),
			Version:     1, CreatedAt: now, UpdatedAt: now,
		},
		warnings: []model.ImportIssue{},
	}
	reviewedBy := model.ImportReviewSource

	categories := imports.CategoryPlan(spec)
	categoryIDs := map[string]string{}
	realmSeen := false
	for index, category := range categories {
		display, key, err := model.NormalizeKey(category.Name, "name")
		if err != nil {
			continue
		}
		id := model.NewID()
		isBuiltin := category.Name == model.BuiltinRealmCategory && !realmSeen
		if isBuiltin {
			realmSeen = true
		}
		categoryIDs[category.Name] = id
		plan.categories = append(plan.categories, model.SystemCategory{
			ID: id, WorkID: workID, Name: display, NameKey: key,
			CategoryType: model.CategoryTypeNormal, IsBuiltin: isBuiltin, IsEnabled: true,
			SortOrder: index, Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}

	nodeIDs := map[string]string{}
	sortCounters := map[string]int{}
	for _, node := range spec.Nodes {
		categoryID := categoryIDs[node.Category]
		if categoryID == "" {
			continue
		}
		id := model.NewID()
		var parentID *string
		if node.Parent != "" {
			value := nodeIDs[nodeKey(node.Category, "", node.Parent)]
			if value == "" {
				continue
			}
			parentID = &value
		}
		sortKey := node.Category + "\x00" + node.Parent
		sortOrder := sortCounters[sortKey]
		sortCounters[sortKey] = sortOrder + 1
		display, key, err := model.NormalizeKey(node.Name, "name")
		if err != nil {
			continue
		}
		nodeIDs[nodeKey(node.Category, node.Parent, node.Name)] = id
		plan.nodes = append(plan.nodes, model.SystemNode{
			ID: id, WorkID: workID, CategoryID: categoryID, ParentID: parentID,
			Name: display, NameKey: key, SortOrder: sortOrder,
			ReviewStatus: model.ReviewStatusPending, ReviewedBy: &reviewedBy,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}

	factionIDs := map[string]string{}
	factionNames := map[string]string{}
	parentOf := map[string]string{}
	for _, faction := range spec.Factions {
		key := strings.ToLower(strings.TrimSpace(faction.Name))
		if _, ok := factionIDs[key]; ok {
			continue
		}
		id := model.NewID()
		factionIDs[key] = id
		factionNames[key] = faction.Name
		parentOf[key] = strings.ToLower(strings.TrimSpace(faction.Parent))
	}
	factions := make([]model.Faction, 0, len(spec.Factions))
	for _, faction := range spec.Factions {
		key := strings.ToLower(strings.TrimSpace(faction.Name))
		id := factionIDs[key]
		if id == "" {
			continue
		}
		var parentID *string
		if parent := parentOf[key]; parent != "" {
			value := factionIDs[parent]
			if value == "" {
				continue
			}
			parentID = &value
		}
		factions = append(factions, model.Faction{
			ID: id, WorkID: workID, Name: factionNames[key], NameKey: key,
			ParentID: parentID, Description: trimmed(faction.Description),
			ReviewStatus: model.ReviewStatusPending, ReviewedBy: &reviewedBy,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}
	sortFactionsByDepth(factions, parentOf)
	plan.factions = factions

	locationIDs := map[string]string{}
	locationNames := map[string]string{}
	locationParents := map[string]string{}
	for _, location := range spec.Locations {
		key := strings.ToLower(strings.TrimSpace(location.Name))
		if _, ok := locationIDs[key]; ok {
			continue
		}
		locationIDs[key] = model.NewID()
		locationNames[key] = location.Name
		locationParents[key] = strings.ToLower(strings.TrimSpace(location.Parent))
	}
	locations := make([]model.Location, 0, len(spec.Locations))
	for _, location := range spec.Locations {
		key := strings.ToLower(strings.TrimSpace(location.Name))
		id := locationIDs[key]
		if id == "" {
			continue
		}
		var parentID *string
		if parent := locationParents[key]; parent != "" {
			value := locationIDs[parent]
			if value == "" {
				continue
			}
			parentID = &value
		}
		locations = append(locations, model.Location{
			ID: id, WorkID: workID, Name: locationNames[key], NameKey: key,
			ParentID: parentID, TypeText: trimmed(location.TypeText), Description: trimmed(location.Description),
			ReviewStatus: model.ReviewStatusPending, ReviewedBy: &reviewedBy,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}
	sortLocationsByDepth(locations, locationParents)
	plan.locations = locations

	characterIDs := map[string]string{}
	for _, character := range spec.Characters {
		key := strings.ToLower(strings.TrimSpace(character.Name))
		if _, ok := characterIDs[key]; ok {
			continue
		}
		characterIDs[key] = model.NewID()
	}
	for _, character := range spec.Characters {
		key := strings.ToLower(strings.TrimSpace(character.Name))
		id := characterIDs[key]
		if id == "" {
			continue
		}
		var factionID *string
		if character.Faction != "" {
			value := factionIDs[strings.ToLower(strings.TrimSpace(character.Faction))]
			if value == "" {
				continue
			}
			factionID = &value
		}
		plan.characters = append(plan.characters, model.Character{
			ID: id, WorkID: workID, Name: character.Name, NameKey: key,
			FactionID: factionID, Description: trimmed(character.Description),
			ReviewStatus: model.ReviewStatusPending, ReviewedBy: &reviewedBy,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
		appendCharacterRelations(plan, workID, id, character, categoryIDs, locationIDs, nodeIDs, now)
	}

	for _, faction := range spec.Factions {
		key := strings.ToLower(strings.TrimSpace(faction.Name))
		if faction.Leader == "" {
			continue
		}
		leaderID := characterIDs[strings.ToLower(strings.TrimSpace(faction.Leader))]
		if leaderID == "" {
			continue
		}
		for index := range plan.factions {
			if plan.factions[index].ID == factionIDs[key] {
				plan.factions[index].LeaderCharacterID = &leaderID
			}
		}
	}

	for _, location := range spec.Locations {
		key := strings.ToLower(strings.TrimSpace(location.Name))
		locationID := locationIDs[key]
		if locationID == "" {
			continue
		}
		for _, relation := range location.Factions {
			factionID := factionIDs[strings.ToLower(strings.TrimSpace(relation.Name))]
			if factionID == "" {
				continue
			}
			plan.locationFactions = append(plan.locationFactions, model.LocationFaction{
				ID: model.NewID(), WorkID: workID, LocationID: locationID, FactionID: factionID,
				Note: trimmed(relation.Note), CreatedAt: now, UpdatedAt: now,
			})
		}
	}

	for _, ranking := range spec.Rankings {
		key := strings.ToLower(strings.TrimSpace(ranking.Name))
		id := model.NewID()
		seatCount := ranking.SeatCount
		if seatCount == 0 {
			seatCount = maxSeatNo(ranking.Seats)
		}
		plan.rankings = append(plan.rankings, model.Ranking{
			ID: id, WorkID: workID, Name: ranking.Name, NameKey: key, SeatCount: seatCount,
			ReviewStatus: model.ReviewStatusPending, ReviewedBy: &reviewedBy,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
		bySeat := map[int]imports.SeatSpec{}
		for _, seat := range ranking.Seats {
			bySeat[seat.No] = seat
		}
		for seatNo := 1; seatNo <= seatCount; seatNo++ {
			row := model.RankingSeat{
				ID: model.NewID(), WorkID: workID, RankingID: id, SeatNo: seatNo,
				Version: 1, CreatedAt: now, UpdatedAt: now,
			}
			if seat, ok := bySeat[seatNo]; ok {
				row.Note = trimmed(seat.Note)
				if characterID := characterIDs[strings.ToLower(strings.TrimSpace(seat.Character))]; characterID != "" {
					row.CharacterID = &characterID
				}
			}
			plan.seats = append(plan.seats, row)
		}
	}

	sortCounters = map[string]int{}
	for _, chronicle := range spec.Chronicles {
		ownerType := model.ChronicleOwnerCharacter
		ownerKey := strings.ToLower(strings.TrimSpace(chronicle.OwnerName))
		ownerID := characterIDs[ownerKey]
		if chronicle.OwnerType == "势力" || chronicle.OwnerType == model.ChronicleOwnerFaction {
			ownerType = model.ChronicleOwnerFaction
			ownerID = factionIDs[ownerKey]
		}
		if ownerID == "" {
			continue
		}
		sortKey := ownerType + "\x00" + ownerID
		sortOrder := sortCounters[sortKey]
		sortCounters[sortKey] = sortOrder + 1
		plan.chronicles = append(plan.chronicles, model.Chronicle{
			ID: model.NewID(), WorkID: workID, OwnerType: ownerType, OwnerID: ownerID,
			Text: chronicle.Text, TimeLabel: trimmed(chronicle.TimeLabel), SortOrder: sortOrder,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		})
	}

	for key, asset := range media {
		characterID := characterIDs[strings.ToLower(strings.TrimSpace(key))]
		if characterID == "" {
			continue
		}
		asset.EntityID = characterID
		asset.CreatedAt = now
		asset.UpdatedAt = now
		plan.media = append(plan.media, asset)
	}
	return plan
}

func appendCharacterRelations(plan *importPlan, workID, characterID string, character imports.CharacterSpec, categoryIDs, locationIDs, nodeIDs map[string]string, now time.Time) {
	for index, alias := range character.Aliases {
		display, key, err := model.NormalizeCharacterNameValue(alias, "alias")
		if err != nil {
			continue
		}
		plan.names = append(plan.names, model.CharacterName{
			ID: model.NewID(), CharacterID: characterID, WorkID: workID,
			Kind: model.CharacterNameKindAlias, Value: display, ValueKey: key,
			SortOrder: index, CreatedAt: now, UpdatedAt: now,
		})
	}
	for index, title := range character.Titles {
		display, key, err := model.NormalizeCharacterNameValue(title, "title")
		if err != nil {
			continue
		}
		plan.names = append(plan.names, model.CharacterName{
			ID: model.NewID(), CharacterID: characterID, WorkID: workID,
			Kind: model.CharacterNameKindTitle, Value: display, ValueKey: key,
			SortOrder: index, CreatedAt: now, UpdatedAt: now,
		})
	}
	for category, nodePath := range character.Extras {
		categoryID := categoryIDs[category]
		if categoryID == "" {
			continue
		}
		parts := strings.Split(nodePath, ">")
		parent := ""
		nodeName := strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			parent = strings.TrimSpace(parts[0])
			nodeName = strings.TrimSpace(parts[1])
		}
		nodeID := nodeIDs[nodeKey(category, parent, nodeName)]
		if nodeID == "" {
			continue
		}
		plan.refs = append(plan.refs, model.CharacterSystemRef{
			ID: model.NewID(), WorkID: workID, CharacterID: characterID,
			CategoryID: categoryID, NodeID: nodeID,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	for _, relation := range character.Locations {
		locationID := locationIDs[strings.ToLower(strings.TrimSpace(relation.Name))]
		if locationID == "" {
			continue
		}
		plan.charLocations = append(plan.charLocations, model.CharacterLocation{
			ID: model.NewID(), WorkID: workID, CharacterID: characterID, LocationID: locationID,
			Note: trimmed(relation.Note), CreatedAt: now, UpdatedAt: now,
		})
	}
}

// writePlan 在单事务内按外键顺序写入全部实体，并重建检索与切换当前作品。
func (s *Import) writePlan(ctx context.Context, plan *importPlan, importID, snapshotID, reportJSON string, finished time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		works := repository.NewWorks(tx)
		if taken, err := works.NameTaken(ctx, plan.work.NameKey); err != nil {
			return err
		} else if taken {
			return model.ErrNameConflict
		}
		if err := works.Insert(ctx, &plan.work); err != nil {
			if repository.IsUniqueViolation(err, "works_name_key_uq") {
				return model.ErrNameConflict
			}
			return err
		}
		if err := createAll(tx, &plan.categories, len(plan.categories)); err != nil {
			return err
		}
		for index := range plan.nodes {
			if err := tx.Create(&plan.nodes[index]).Error; err != nil {
				return err
			}
		}
		for index := range plan.factions {
			if err := tx.Create(&plan.factions[index]).Error; err != nil {
				return err
			}
		}
		for index := range plan.locations {
			if err := tx.Create(&plan.locations[index]).Error; err != nil {
				return err
			}
		}
		if err := createAll(tx, &plan.characters, len(plan.characters)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.names, len(plan.names)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.refs, len(plan.refs)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.charLocations, len(plan.charLocations)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.locationFactions, len(plan.locationFactions)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.rankings, len(plan.rankings)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.seats, len(plan.seats)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.chronicles, len(plan.chronicles)); err != nil {
			return err
		}
		if err := createAll(tx, &plan.media, len(plan.media)); err != nil {
			return err
		}
		for index := range plan.factions {
			if plan.factions[index].LeaderCharacterID == nil {
				continue
			}
			if err := tx.Model(&model.Faction{}).Where("id = ?", plan.factions[index].ID).
				Update("leader_character_id", *plan.factions[index].LeaderCharacterID).Error; err != nil {
				return err
			}
		}
		if err := search.NewRefresher(tx).RefreshWork(ctx, plan.work.ID); err != nil {
			return err
		}
		if err := repository.NewSettings(tx).SetCurrentWork(ctx, &plan.work.ID, plan.work.CreatedAt); err != nil {
			return err
		}
		return repository.NewImports(tx).Update(ctx, importID, map[string]any{
			"status":      model.ImportStatusCompleted,
			"stage":       model.ImportStageCompleted,
			"work_id":     plan.work.ID,
			"snapshot_id": snapshotID,
			"report_json": reportJSON,
			"error_count": 0,
			"error_code":  nil,
			"finished_at": finished,
			"updated_at":  finished,
		})
	})
}

func createAll(tx *gorm.DB, value any, count int) error {
	if count == 0 {
		return nil
	}
	return tx.Create(value).Error
}

func nodeKey(category, parent, name string) string {
	return category + "\x00" + parent + "\x00" + name
}

func trimmed(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func maxSeatNo(seats []imports.SeatSpec) int {
	max := 0
	for _, seat := range seats {
		if seat.No > max {
			max = seat.No
		}
	}
	if max == 0 {
		max = model.DefaultRankingSeats
	}
	return max
}

func toModelIssues(items []imports.Issue) []model.ImportIssue {
	result := make([]model.ImportIssue, 0, len(items))
	for _, item := range items {
		result = append(result, model.ImportIssue{Line: item.Line, Field: item.Field, Code: item.Code, Message: item.Message})
	}
	return result
}

// sortFactionsByDepth 让自引用外键的记录按层级稳定排序，父级先写入。
func sortFactionsByDepth(items []model.Faction, parentOf map[string]string) {
	sort.SliceStable(items, func(i, j int) bool {
		return depthOf(strings.ToLower(items[i].Name), parentOf) < depthOf(strings.ToLower(items[j].Name), parentOf)
	})
}

func sortLocationsByDepth(items []model.Location, parentOf map[string]string) {
	sort.SliceStable(items, func(i, j int) bool {
		return depthOf(strings.ToLower(items[i].Name), parentOf) < depthOf(strings.ToLower(items[j].Name), parentOf)
	})
}

func depthOf(name string, parentOf map[string]string) int {
	current := name
	level := 0
	seen := map[string]bool{}
	for current != "" && !seen[current] && level <= len(parentOf) {
		seen[current] = true
		level++
		current = parentOf[current]
	}
	return level
}
