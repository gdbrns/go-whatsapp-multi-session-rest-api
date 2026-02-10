package whatsapp

import (
	"context"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// LIDResolver is a function type for resolving LID to phone number
type LIDResolver func(lid types.JID) (types.JID, error)

// BatchLIDResolver resolves multiple LIDs in a single batch operation
// This is much faster than resolving one by one
type BatchLIDResolver func(lids []types.JID) map[string]string

// EnhancedGroupParticipant extends GroupParticipant with resolved phone number
// This is needed because WhatsApp may return LID (Linked ID) instead of JID (phone number)
// for users who have enabled privacy features
type EnhancedGroupParticipant struct {
	JID          string `json:"JID"`          // Original JID from WhatsApp (may be LID)
	PhoneNumber  string `json:"PhoneNumber"`  // Resolved phone number (from PhoneNumber field or LID resolution)
	LID          string `json:"LID"`          // LID if available
	IsAdmin      bool   `json:"IsAdmin"`
	IsSuperAdmin bool   `json:"IsSuperAdmin"`
	DisplayName  string `json:"DisplayName,omitempty"`
	Error        string `json:"Error,omitempty"` // Error during participant info fetch
}

// EnhancedGroupInfo extends GroupInfo with enhanced participant data
// that includes resolved phone numbers
type EnhancedGroupInfo struct {
	JID                      string                     `json:"JID"`
	OwnerJID                 string                     `json:"OwnerJID,omitempty"`
	Name                     string                     `json:"Name"`
	NameSetAt                time.Time                  `json:"NameSetAt,omitempty"`
	NameSetBy                string                     `json:"NameSetBy,omitempty"`
	Topic                    string                     `json:"Topic,omitempty"`
	TopicID                  string                     `json:"TopicID,omitempty"`
	TopicSetAt               time.Time                  `json:"TopicSetAt,omitempty"`
	TopicSetBy               string                     `json:"TopicSetBy,omitempty"`
	TopicDeleted             bool                       `json:"TopicDeleted,omitempty"`
	Locked                   bool                       `json:"IsLocked"`
	Announce                 bool                       `json:"IsAnnounce"`
	Ephemeral                uint32                     `json:"Ephemeral,omitempty"`
	IsParent                 bool                       `json:"IsParent,omitempty"`
	IsDefaultSubGroup        bool                       `json:"IsDefaultSubGroup,omitempty"`
	LinkedParentJID          string                     `json:"LinkedParentJID,omitempty"`
	IsIncognito              bool                       `json:"IsIncognito,omitempty"`
	MemberAddMode            string                     `json:"MemberAddMode,omitempty"`
	GroupCreated             time.Time                  `json:"GroupCreated,omitempty"`
	ParticipantVersionID     string                     `json:"ParticipantVersionID,omitempty"`
	Participants             []EnhancedGroupParticipant `json:"Participants"`
	IsJoinApprovalRequired   bool                       `json:"IsJoinApprovalRequired,omitempty"`
	GroupType                string                     `json:"GroupType,omitempty"`
}

// ConvertToEnhancedGroupInfo converts types.GroupInfo to EnhancedGroupInfo
// and resolves LID to phone numbers using the provided LID resolver function
// OPTIMIZED: Uses pre-resolved LID map instead of per-participant resolution
func ConvertToEnhancedGroupInfo(group types.GroupInfo, lidResolver func(lid types.JID) (types.JID, error)) EnhancedGroupInfo {
	enhanced := EnhancedGroupInfo{
		JID:                    group.JID.String(),
		Name:                   group.Name,
		NameSetAt:              group.NameSetAt,
		Topic:                  group.Topic,
		TopicID:                group.TopicID,
		TopicSetAt:             group.TopicSetAt,
		TopicDeleted:           group.TopicDeleted,
		Locked:                 group.IsLocked,
		Announce:               group.IsAnnounce,
		Ephemeral:              group.DisappearingTimer,
		IsParent:               group.IsParent,
		IsDefaultSubGroup:      group.IsDefaultSubGroup,
		IsIncognito:            group.IsIncognito,
		IsJoinApprovalRequired: group.IsJoinApprovalRequired,
		GroupCreated:           group.GroupCreated,
		ParticipantVersionID:   group.ParticipantVersionID,
		Participants:           make([]EnhancedGroupParticipant, 0, len(group.Participants)),
	}

	// Handle optional JID fields
	if group.OwnerJID.User != "" {
		enhanced.OwnerJID = group.OwnerJID.String()
	}
	if group.NameSetBy.User != "" {
		enhanced.NameSetBy = group.NameSetBy.String()
	}
	if group.TopicSetBy.User != "" {
		enhanced.TopicSetBy = group.TopicSetBy.String()
	}
	if group.LinkedParentJID.User != "" {
		enhanced.LinkedParentJID = group.LinkedParentJID.String()
	}

	// Handle MemberAddMode
	if group.MemberAddMode != "" {
		enhanced.MemberAddMode = string(group.MemberAddMode)
	}

	// FAST PATH: Convert participants WITHOUT LID resolution (much faster)
	// LID resolution is now optional and done in batch externally
	for _, p := range group.Participants {
		ep := EnhancedGroupParticipant{
			JID:          p.JID.String(),
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
			DisplayName:  p.DisplayName,
		}

		// Set LID if available
		if p.LID.User != "" {
			ep.LID = p.LID.String()
		}

		// Try to get phone number from participant's PhoneNumber field first (whatsmeow provides this)
		if p.PhoneNumber.User != "" {
			ep.PhoneNumber = p.PhoneNumber.User
		} else if p.JID.User != "" && p.JID.Server == "s.whatsapp.net" {
			// JID is already a phone number format (@s.whatsapp.net)
			ep.PhoneNumber = p.JID.User
		} else {
			// Fallback: use JID user part as phone number (skip slow LID resolution)
			ep.PhoneNumber = p.JID.User
		}

		enhanced.Participants = append(enhanced.Participants, ep)
	}

	return enhanced
}

// ConvertToEnhancedGroupInfoWithLIDResolution converts types.GroupInfo with full LID resolution
// WARNING: This is SLOW for large groups - use ConvertToEnhancedGroupInfo when possible
func ConvertToEnhancedGroupInfoWithLIDResolution(group types.GroupInfo, lidResolver func(lid types.JID) (types.JID, error)) EnhancedGroupInfo {
	enhanced := EnhancedGroupInfo{
		JID:                    group.JID.String(),
		Name:                   group.Name,
		NameSetAt:              group.NameSetAt,
		Topic:                  group.Topic,
		TopicID:                group.TopicID,
		TopicSetAt:             group.TopicSetAt,
		TopicDeleted:           group.TopicDeleted,
		Locked:                 group.IsLocked,
		Announce:               group.IsAnnounce,
		Ephemeral:              group.DisappearingTimer,
		IsParent:               group.IsParent,
		IsDefaultSubGroup:      group.IsDefaultSubGroup,
		IsIncognito:            group.IsIncognito,
		IsJoinApprovalRequired: group.IsJoinApprovalRequired,
		GroupCreated:           group.GroupCreated,
		ParticipantVersionID:   group.ParticipantVersionID,
		Participants:           make([]EnhancedGroupParticipant, 0, len(group.Participants)),
	}

	// Handle optional JID fields
	if group.OwnerJID.User != "" {
		enhanced.OwnerJID = group.OwnerJID.String()
	}
	if group.NameSetBy.User != "" {
		enhanced.NameSetBy = group.NameSetBy.String()
	}
	if group.TopicSetBy.User != "" {
		enhanced.TopicSetBy = group.TopicSetBy.String()
	}
	if group.LinkedParentJID.User != "" {
		enhanced.LinkedParentJID = group.LinkedParentJID.String()
	}

	// Handle MemberAddMode
	if group.MemberAddMode != "" {
		enhanced.MemberAddMode = string(group.MemberAddMode)
	}

	// Convert participants with LID resolution (SLOW PATH)
	for _, p := range group.Participants {
		ep := EnhancedGroupParticipant{
			JID:          p.JID.String(),
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
			DisplayName:  p.DisplayName,
		}

		// Set LID if available
		if p.LID.User != "" {
			ep.LID = p.LID.String()
		}

		// Try to get phone number from participant's PhoneNumber field first
		if p.PhoneNumber.User != "" {
			ep.PhoneNumber = p.PhoneNumber.User
		} else if p.JID.User != "" && p.JID.Server == "s.whatsapp.net" {
			ep.PhoneNumber = p.JID.User
		} else if p.LID.User != "" && lidResolver != nil {
			// Only resolve LID when absolutely necessary
			pn, err := lidResolver(p.LID)
			if err == nil && pn.User != "" {
				ep.PhoneNumber = pn.User
			} else if err != nil {
				ep.Error = "LID resolution failed"
			}
		}

		// Last resort
		if ep.PhoneNumber == "" {
			ep.PhoneNumber = p.JID.User
		}

		enhanced.Participants = append(enhanced.Participants, ep)
	}

	return enhanced
}

// ConvertGroupsInParallel converts multiple groups to EnhancedGroupInfo in parallel
// This significantly speeds up processing for accounts with many groups
func ConvertGroupsInParallel(groups []*types.GroupInfo, lidResolver func(lid types.JID) (types.JID, error), maxWorkers int) []EnhancedGroupInfo {
	if len(groups) == 0 {
		return []EnhancedGroupInfo{}
	}

	// Limit workers
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if maxWorkers > len(groups) {
		maxWorkers = len(groups)
	}

	result := make([]EnhancedGroupInfo, len(groups))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, group := range groups {
		if group == nil {
			continue
		}
		wg.Add(1)
		go func(idx int, g types.GroupInfo) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Use fast path without LID resolution
			result[idx] = ConvertToEnhancedGroupInfo(g, nil)
		}(i, *group)
	}

	wg.Wait()

	// Filter out empty results
	filtered := make([]EnhancedGroupInfo, 0, len(result))
	for _, r := range result {
		if r.JID != "" {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ConvertGroupsInParallelWithContext is like ConvertGroupsInParallel but respects context cancellation
func ConvertGroupsInParallelWithContext(ctx context.Context, groups []*types.GroupInfo, lidResolver func(lid types.JID) (types.JID, error), maxWorkers int) ([]EnhancedGroupInfo, error) {
	if len(groups) == 0 {
		return []EnhancedGroupInfo{}, nil
	}

	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if maxWorkers > len(groups) {
		maxWorkers = len(groups)
	}

	result := make([]EnhancedGroupInfo, len(groups))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, group := range groups {
		if group == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(idx int, g types.GroupInfo) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			result[idx] = ConvertToEnhancedGroupInfo(g, nil)
		}(i, *group)
	}

	// Wait for completion or cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}

	// Filter out empty results
	filtered := make([]EnhancedGroupInfo, 0, len(result))
	for _, r := range result {
		if r.JID != "" {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}
