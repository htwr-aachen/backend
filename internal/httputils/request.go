package httputils

import (
	"fmt"
	"net/http"
	"strconv"
)

func PathId(r *http.Request, accessor string) (uint32, error) {
	str := r.PathValue(accessor)
	idBig, err := strconv.ParseUint(str, 10, 32)
	return uint32(idBig), err
}

// getPaginationParams extracts and validates pagination parameters from the request.
// It returns 0,0,nil on empty or incomplete pagination args, or an error if parsing fails.
func GetPaginationParams(r *http.Request, required bool) (lastPriority int, lastId uint32, limit int, err error) {
	lastPriorityStr := r.URL.Query().Get("last-priority")
	if required && lastPriorityStr == "" {
		return 0, 0, 0, nil
	}

	if lastPriorityStr != "" {
		lastPriority, err = strconv.Atoi(lastPriorityStr)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("could not convert last-priority pagination param: %w", err)
		}
	}

	lastIdStr := r.URL.Query().Get("last-id")
	if required && lastIdStr == "" {
		return 0, 0, 0, nil
	}

	if lastIdStr != "" {
		beforeIdBig, err := strconv.ParseUint(lastIdStr, 10, 32)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("could not convert last-id pagination param: %w", err)
		}

		lastId = uint32(beforeIdBig)
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("could not convert limit pagination param: %w", err)
		}
	}

	if limit <= 0 {
		return 0, 0, 0, fmt.Errorf("limit pagination param must be >= 1")
	}
	return lastPriority, lastId, limit, nil
}
