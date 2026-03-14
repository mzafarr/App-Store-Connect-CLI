package googleplay

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const productionTrackName = "production"

func requireProductionConfirmation(track string, confirmed bool) error {
	if strings.EqualFold(strings.TrimSpace(track), productionTrackName) && !confirmed {
		fmt.Fprintln(os.Stderr, "Error: --confirm-production is required when --track is production")
		return flag.ErrHelp
	}
	return nil
}
