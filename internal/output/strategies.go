// Package output provides the Writer interface and backend implementations (CSV, SQLite)
// for persisting Mist API response data. All endpoint primary-key strategies are defined
// here, ported verbatim from Python's ENDPOINT_PRIMARY_KEY_STRATEGIES dict.
package output

// PKType identifies which primary-key strategy to apply when writing rows to SQLite.
type PKType string

const (
	PKTypeNatural       PKType = "natural_pk"                // UUID from API (e.g. "id")
	PKTypeComposite     PKType = "composite_pk"              // Multi-column PK for time-series
	PKTypeAutoIncrement PKType = "auto_increment_with_unique" // No stable key -- use rowid
)

// EndpointStrategy describes how rows are keyed and indexed for a given API endpoint.
type EndpointStrategy struct {
	Type        PKType   // Determines upsert vs insert-or-ignore behaviour
	PrimaryKey  []string // Column(s) forming the unique key for deduplication
	Indexes     []string // Additional columns to index for query performance
	Description string   // Human-readable note so NOC engineers understand the strategy
}

// Strategies maps each known API endpoint name to its primary-key strategy.
// Ported verbatim from Python's ENDPOINT_PRIMARY_KEY_STRATEGIES dict.
// Natural PK entries use a stable UUID; composite PKs combine columns for time-series;
// auto-increment entries have no stable key and use SQLite rowid.
var Strategies = map[string]EndpointStrategy{

	// ── Natural PK entries ──────────────────────────────────────────────────
	// Each entity has a stable API-provided UUID that uniquely identifies it.

	"listOrgSites": {
		Type:        PKTypeNatural,                              // Stable site UUID from Mist API
		PrimaryKey:  []string{"id"},                            // Single UUID PK
		Indexes:     []string{"org_id", "name", "country_code"}, // Common query filters
		Description: "Organisation site list -- one row per site UUID",
	},
	"listOrgDevices": {
		Type:        PKTypeNatural,                        // Stable device UUID from Mist API
		PrimaryKey:  []string{"id"},                      // Device UUID PK
		Indexes:     []string{"site_id", "mac", "type", "model"}, // Frequently filtered columns
		Description: "Organisation device inventory -- one row per device UUID",
	},
	"listOrgWlans": {
		Type:        PKTypeNatural,                  // Stable WLAN UUID from Mist API
		PrimaryKey:  []string{"id"},                // WLAN UUID PK
		Indexes:     []string{"org_id", "ssid"},    // SSID is the human-readable identifier
		Description: "Organisation WLAN list -- one row per WLAN UUID",
	},
	"listOrgNetworks": {
		Type:        PKTypeNatural,                  // Stable network UUID from Mist API
		PrimaryKey:  []string{"id"},                // Network UUID PK
		Indexes:     []string{"org_id", "name"},    // Name is the primary human identifier
		Description: "Organisation network/VLAN list -- one row per network UUID",
	},
	"listOrgGatewayTemplates": {
		Type:        PKTypeNatural,                  // Stable template UUID from Mist API
		PrimaryKey:  []string{"id"},                // Template UUID PK
		Indexes:     []string{"org_id", "name"},    // Name identifies the template
		Description: "Organisation gateway template list -- one row per template UUID",
	},
	"listOrgRfTemplates": {
		Type:        PKTypeNatural,                  // Stable RF template UUID from Mist API
		PrimaryKey:  []string{"id"},                // RF template UUID PK
		Indexes:     []string{"org_id", "name"},    // Name identifies the RF template
		Description: "Organisation RF template list -- one row per RF template UUID",
	},
	"listOrgNetworkTemplates": {
		Type:        PKTypeNatural,                  // Stable network template UUID from Mist API
		PrimaryKey:  []string{"id"},                // Network template UUID PK
		Indexes:     []string{"org_id", "name"},    // Name identifies the network template
		Description: "Organisation network template list -- one row per template UUID",
	},
	"listOrgDeviceProfiles": {
		Type:        PKTypeNatural,                  // Stable device profile UUID from Mist API
		PrimaryKey:  []string{"id"},                // Device profile UUID PK
		Indexes:     []string{"org_id", "name"},    // Name identifies the device profile
		Description: "Organisation device profile list -- one row per profile UUID",
	},
	"listSiteDeviceUpgrades": {
		Type:        PKTypeNatural,              // Stable upgrade job UUID from Mist API
		PrimaryKey:  []string{"id"},            // Upgrade job UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every upgrade job
		Description: "Site device upgrade list -- one row per upgrade job UUID",
	},
	"listSiteWlansDerived": {
		Type:        PKTypeNatural,                   // Stable WLAN UUID from derived config
		PrimaryKey:  []string{"id"},                 // WLAN UUID PK
		Indexes:     []string{"site_id", "ssid"},    // Site and SSID are common filters
		Description: "Site derived WLAN list -- one row per WLAN UUID",
	},
	"listSiteNetworksDerived": {
		Type:        PKTypeNatural,                   // Stable network UUID from derived config
		PrimaryKey:  []string{"id"},                 // Network UUID PK
		Indexes:     []string{"site_id", "name"},    // Site and name are common filters
		Description: "Site derived network list -- one row per network UUID",
	},
	"listSiteVpnsDerived": {
		Type:        PKTypeNatural,              // Stable VPN UUID from derived config
		PrimaryKey:  []string{"id"},            // VPN UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every VPN definition
		Description: "Site derived VPN list -- one row per VPN UUID",
	},
	"listSiteServicesDerived": {
		Type:        PKTypeNatural,                   // Stable service UUID from derived config
		PrimaryKey:  []string{"id"},                 // Service UUID PK
		Indexes:     []string{"site_id", "name"},    // Site and name are common filters
		Description: "Site derived services list -- one row per service UUID",
	},
	"listSiteServicePoliciesDerived": {
		Type:        PKTypeNatural,              // Stable policy UUID from derived config
		PrimaryKey:  []string{"id"},            // Service policy UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every policy
		Description: "Site derived service policies -- one row per policy UUID",
	},
	"listSiteGatewayTemplatesDerived": {
		Type:        PKTypeNatural,              // Stable template UUID from derived config
		PrimaryKey:  []string{"id"},            // Gateway template UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every template
		Description: "Site derived gateway templates -- one row per template UUID",
	},
	"listSiteSiteTemplatesDerived": {
		Type:        PKTypeNatural,              // Stable site-template UUID from derived config
		PrimaryKey:  []string{"id"},            // Site template UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every site template
		Description: "Site derived site templates -- one row per template UUID",
	},
	"listSiteDeviceProfilesDerived": {
		Type:        PKTypeNatural,              // Stable device profile UUID from derived config
		PrimaryKey:  []string{"id"},            // Device profile UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every device profile
		Description: "Site derived device profiles -- one row per profile UUID",
	},
	"listSiteIdpProfilesDerived": {
		Type:        PKTypeNatural,              // Stable IDP profile UUID from derived config
		PrimaryKey:  []string{"id"},            // IDP profile UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every IDP profile
		Description: "Site derived IDP profiles -- one row per profile UUID",
	},
	"listSiteAllGuestAuthorizationsDerived": {
		Type:        PKTypeNatural,              // MAC address is the natural key for guests
		PrimaryKey:  []string{"mac"},           // Guest MAC as the unique identifier
		Indexes:     []string{"site_id"},       // Site scopes every guest authorization
		Description: "Site derived guest authorizations -- one row per guest MAC address",
	},
	"listSiteApTemplatesDerived": {
		Type:        PKTypeNatural,              // Stable AP template UUID from derived config
		PrimaryKey:  []string{"id"},            // AP template UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every AP template
		Description: "Site derived AP templates -- one row per AP template UUID",
	},
	"listSiteRfTemplatesDerived": {
		Type:        PKTypeNatural,              // Stable RF template UUID from derived config
		PrimaryKey:  []string{"id"},            // RF template UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every RF template
		Description: "Site derived RF templates -- one row per RF template UUID",
	},
	"listSiteAAMWProfilesDerived": {
		Type:        PKTypeNatural,              // Stable AAMW profile UUID from derived config
		PrimaryKey:  []string{"id"},            // AAMW profile UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every AAMW profile
		Description: "Site derived advanced anti-malware profiles -- one row per profile UUID",
	},
	"listSiteAntivirusProfilesDerived": {
		Type:        PKTypeNatural,              // Stable antivirus profile UUID from derived config
		PrimaryKey:  []string{"id"},            // Antivirus profile UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every antivirus profile
		Description: "Site derived antivirus profiles -- one row per profile UUID",
	},
	"listSiteSecIntelProfilesDerived": {
		Type:        PKTypeNatural,              // Stable sec-intel profile UUID from derived config
		PrimaryKey:  []string{"id"},            // Security intelligence profile UUID PK
		Indexes:     []string{"site_id"},       // Site scopes every sec-intel profile
		Description: "Site derived security intelligence profiles -- one row per profile UUID",
	},
	"generateE911BSSIDReport": {
		Type:        PKTypeNatural,              // BSSID is the natural key for E911 entries
		PrimaryKey:  []string{"bssid"},         // BSSID uniquely identifies an E911 radio entry
		Indexes:     []string{"site_id"},       // Site scopes every BSSID entry
		Description: "E911 BSSID report -- one row per BSSID",
	},
	"sitesMissingInfrastructure": {
		Type:        PKTypeNatural,              // site_id is the natural key here
		PrimaryKey:  []string{"site_id"},       // Site UUID is the unique key for this report
		Indexes:     []string{"org_id"},        // Org scopes the report
		Description: "Sites with missing infrastructure -- one row per site UUID",
	},
	"sitesWithOfflineInfrastructure": {
		Type:        PKTypeNatural,              // site_id is the natural key here
		PrimaryKey:  []string{"site_id"},       // Site UUID is the unique key for this report
		Indexes:     []string{"org_id"},        // Org scopes the report
		Description: "Sites with offline infrastructure -- one row per site UUID",
	},

	// ── Composite PK entries ────────────────────────────────────────────────
	// Time-series data with no single stable UUID; combined columns form a unique key.

	"searchOrgDeviceEvents": {
		Type:        PKTypeComposite,                                  // Time-series: same event id can repeat at different times
		PrimaryKey:  []string{"id", "device_id", "timestamp"},        // Composite key for deduplication
		Description: "Org device events -- deduplicated on event id + device + time",
	},
	"searchOrgWirelessClients": {
		Type:        PKTypeComposite,                       // Time-series: same client appears across multiple polls
		PrimaryKey:  []string{"mac", "timestamp"},         // MAC + time forms the composite key
		Description: "Org wireless client snapshots -- deduplicated on MAC + timestamp",
	},
	"searchOrgWiredClients": {
		Type:        PKTypeComposite,                       // Time-series: same client appears across multiple polls
		PrimaryKey:  []string{"mac", "timestamp"},         // MAC + time forms the composite key
		Description: "Org wired client snapshots -- deduplicated on MAC + timestamp",
	},
	"globalWiredClientReport": {
		Type:        PKTypeComposite,                       // Cross-site report: same MAC across multiple polls
		PrimaryKey:  []string{"mac", "timestamp"},         // MAC + time deduplicates cross-site entries
		Description: "Global wired client report -- deduplicated on MAC + timestamp",
	},
	"wiredClientManufacturerReport": {
		Type:        PKTypeComposite,                       // Aggregation report: same MAC across multiple polls
		PrimaryKey:  []string{"mac", "timestamp"},         // MAC + time deduplicates per-poll entries
		Description: "Wired client manufacturer report -- deduplicated on MAC + timestamp",
	},
	"tracerouteFromDevice": {
		Type:        PKTypeComposite,                              // Traceroute data changes per run
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates per-run entries
		Description: "Traceroute results from device -- deduplicated on device + timestamp",
	},
	"showSiteGatewayOspfNeighbors": {
		Type:        PKTypeComposite,                              // Neighbour table snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates snapshots
		Description: "Gateway OSPF neighbour table -- deduplicated on device + timestamp",
	},
	"showSiteGatewayOspfInterfaces": {
		Type:        PKTypeComposite,                              // Interface table snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates snapshots
		Description: "Gateway OSPF interface table -- deduplicated on device + timestamp",
	},
	"showSiteGatewayOspfDatabase": {
		Type:        PKTypeComposite,                              // LSDB snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates LSDB entries
		Description: "Gateway OSPF link-state database -- deduplicated on device + timestamp",
	},
	"showSiteGatewayOspfSummary": {
		Type:        PKTypeComposite,                              // Summary snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates summaries
		Description: "Gateway OSPF summary -- deduplicated on device + timestamp",
	},
	"showSiteSsrAndSrxSessions": {
		Type:        PKTypeComposite,                              // Session table snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates session tables
		Description: "SSR/SRX active sessions -- deduplicated on device + timestamp",
	},
	"showSiteSsrServicePath": {
		Type:        PKTypeComposite,                              // Service path snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates path entries
		Description: "SSR service path table -- deduplicated on device + timestamp",
	},
	"showSiteDeviceBgpSummary": {
		Type:        PKTypeComposite,                              // BGP summary snapshot per poll
		PrimaryKey:  []string{"device_id", "timestamp"},          // Device + time deduplicates summaries
		Description: "Device BGP summary -- deduplicated on device + timestamp",
	},

	// ── Auto-increment entries ──────────────────────────────────────────────
	// Aggregated or summary data with no stable natural key; SQLite rowid is used.

	"getOrgLicensesSummary": {
		Type:        PKTypeAutoIncrement,                                   // Summary row with no stable key
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Org licence summary -- auto-increment rowid, no upsert",
	},
	"listOrgLicenses": {
		Type:        PKTypeAutoIncrement,                                   // License list with no stable dedup key
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Org licence list -- auto-increment rowid, no upsert",
	},
	"listSiteDeviceRadioChannels": {
		Type:        PKTypeAutoIncrement,                                   // Radio channel data with no stable key
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Site device radio channels -- auto-increment rowid, no upsert",
	},
	"listSiteRfSpectrumAnalysis": {
		Type:        PKTypeAutoIncrement,                                   // RF spectrum data varies per scan
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Site RF spectrum analysis -- auto-increment rowid, no upsert",
	},
	"listSiteNetworkTemplatesDerived": {
		Type:        PKTypeAutoIncrement,                                   // Derived list with no stable compound key
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Site derived network templates -- auto-increment rowid, no upsert",
	},
	"listSiteUiSettingDerived": {
		Type:        PKTypeAutoIncrement,                                   // UI settings blob with no stable key
		PrimaryKey:  []string{"misthelper_internal_id"},                   // Virtual rowid PK
		Description: "Site derived UI settings -- auto-increment rowid, no upsert",
	},
}

// Get returns the EndpointStrategy for the given endpoint name.
// When the endpoint is not registered, a safe auto-increment default is returned
// so unknown endpoints never cause data loss -- they just append rows.
func Get(endpoint string) EndpointStrategy {
	if strategy, ok := Strategies[endpoint]; ok { // Look up the endpoint in the strategy map
		return strategy // Return the registered strategy for this endpoint
	}
	return EndpointStrategy{ // Default for unknown endpoints -- safe append-only behaviour
		Type:        PKTypeAutoIncrement,                  // Auto-rowid prevents any accidental overwrites
		PrimaryKey:  []string{"misthelper_internal_id"},  // Virtual rowid as PK placeholder
		Description: "Default: auto-increment (endpoint not registered in strategies map)",
	}
}
