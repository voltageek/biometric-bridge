// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"image/color"
)

// Color tokens from the UI design specification
var (
	// Surface colors
	SurfaceDark = color.NRGBA{15, 22, 35, 255}    // #0F1623 (header, dropdown)
	SurfaceMenu = color.NRGBA{30, 37, 53, 255}    // #1E2535 (dropdown background)
	SurfaceBody = color.NRGBA{244, 245, 248, 255} // #F4F5F8 (panel body)
	SurfaceCard = color.NRGBA{255, 255, 255, 255} // #FFFFFF (buttons, cards)
	SurfaceUser = color.NRGBA{234, 237, 250, 255} // #EAEDFA (user card)

	// Accent colors
	BluePrimary = color.NRGBA{30, 58, 138, 255}  // #1E3A8A (headings, button text)
	BlueAccent  = color.NRGBA{59, 130, 246, 255} // #3B82F6 (timestamps)
	GreenActive = color.NRGBA{34, 197, 94, 255}  // #22C55E (active status)
	OrangeWarn  = color.NRGBA{249, 115, 22, 255} // #F97316 (warning events)
	RedDanger   = color.NRGBA{239, 68, 68, 255}  // #EF4444 (stop actions)

	// Text colors
	TextPrimary = color.NRGBA{17, 24, 39, 255}    // #111827 (body text)
	TextMuted   = color.NRGBA{107, 114, 128, 255} // #6B7280 (secondary text)
	TextLabel   = color.NRGBA{138, 143, 160, 255} // #8A8FA0 (section labels)
	White       = color.NRGBA{255, 255, 255, 255} // #FFFFFF (header text)

	// Border colors
	BorderDefault = color.NRGBA{209, 213, 219, 255} // #D1D5DB (button borders)
	BorderHover   = color.NRGBA{191, 219, 254, 255} // #BFDBFE (button hover bg)
)

// Layout constants
const (
	PanelWidth        = 340 // px
	HeaderHeight      = 52  // px
	SectionGap        = 16  // px
	ButtonHeight      = 44  // px
	ButtonGap         = 10  // px
	ListItemPadding   = 8   // px
	CardPadding       = 12  // px
	IconSize          = 16  // px
	AvatarSize        = 40  // px
	BorderRadius      = 8   // px
	BorderRadiusLarge = 12  // px (panel corners)
)

// Typography constants
const (
	FontSizeTitle     = 16 // Header title
	FontSizeHeading   = 20 // Bridge address
	FontSizeLabel     = 10 // Section labels
	FontSizeBody      = 13 // Event titles
	FontSizeSmall     = 12 // Descriptions
	FontSizeTimestamp = 11 // Monospace timestamps
	FontSizeStatus    = 11 // Status badges
)
