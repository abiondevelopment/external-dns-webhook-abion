package dnsprovider

import (
	"context"
	"testing"

	"github.com/abiondevelopment/external-dns-webhook-abion/internal"
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/configuration"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

type zonesResponse struct {
	*internal.APIResponse[[]internal.Zone]
	err error
}
type zoneResponse struct {
	*internal.APIResponse[*internal.Zone]
	err error
}

type patchZoneResponse struct {
	*internal.APIResponse[*internal.Zone]
	err error
}

type mockClient struct {
	getZones  zonesResponse
	getZone   zoneResponse
	patchZone patchZoneResponse
}

func (c mockClient) GetZones(ctx context.Context, page *internal.Pagination) (*internal.APIResponse[[]internal.Zone], error) {
	r := c.getZones
	return r.APIResponse, r.err
}

func (c mockClient) GetZone(ctx context.Context, name string) (*internal.APIResponse[*internal.Zone], error) {
	r := c.getZone
	return r.APIResponse, r.err
}

func (c mockClient) PatchZone(ctx context.Context, name string, patch internal.ZoneRequest) (*internal.APIResponse[*internal.Zone], error) {
	r := c.patchZone
	return r.APIResponse, r.err
}

// checkError checks if an error is thrown when expected.
func checkError(t *testing.T, err error, errExp bool) {
	isErr := err != nil
	if (isErr && !errExp) || (!isErr && errExp) {
		t.Fail()
	}
}

func Test_Records(t *testing.T) {
	type testCase struct {
		name     string
		provider AbionProvider
		expected struct {
			endpoints int
			err       bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := tc.provider.Records(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			assert.Equal(t, tc.expected.endpoints, len(actual))
		}
	}

	testCases := []testCase{
		{
			name: "No records",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  0,
									Total:  0,
								},
							},
							Data: []internal.Zone{},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 0,
			},
		}, {
			name: "Records returned",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  1,
									Total:  1,
								},
							},
							Data: []internal.Zone{
								{
									Type: "zone",
									ID:   "abion.test",
								},
							},
						},
						err: nil,
					},
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.0",
												},
											},
										},
										"www": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.1",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 2,
			},
		}, {
			name: "zones error",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 0,
				err:       true,
			},
		}, {
			name: "zone error",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  1,
									Total:  1,
								},
							},
							Data: []internal.Zone{
								{
									Type: "zone",
									ID:   "abion.test",
								},
							},
						},
						err: nil,
					},
					getZone: zoneResponse{
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				endpoints int
				err       bool
			}{
				endpoints: 0,
				err:       true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_endpointsByZone(t *testing.T) {
	type testCase struct {
		name             string
		provider         AbionProvider
		zoneNameIDMapper provider.ZoneIDName
		endpoints        []*endpoint.Endpoint
		expected         struct {
			keys   int
			values int
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.provider.endpointsByZone(tc.zoneNameIDMapper, tc.endpoints)
		assert.Equal(t, tc.expected.keys, len(actual))

		count := 0
		for _, value := range actual {
			count = count + len(value)
		}
		assert.Equal(t, tc.expected.values, count)
	}

	testCases := []testCase{
		{
			name:             "Empty zone mapper and empty endpoints",
			provider:         AbionProvider{},
			zoneNameIDMapper: provider.ZoneIDName{},
			endpoints:        []*endpoint.Endpoint{},
			expected: struct {
				keys   int
				values int
			}{
				keys:   0,
				values: 0,
			},
		},
		{
			name:             "Empty zone mapper",
			provider:         AbionProvider{},
			zoneNameIDMapper: provider.ZoneIDName{},
			endpoints: []*endpoint.Endpoint{
				{
					DNSName: "abion.test",
				},
			},
			expected: struct {
				keys   int
				values int
			}{
				keys:   0,
				values: 0,
			},
		},
		{
			name:     "endpoint by zone",
			provider: AbionProvider{},
			zoneNameIDMapper: provider.ZoneIDName{
				"abion.test": "abion.test",
			},
			endpoints: []*endpoint.Endpoint{
				{
					DNSName: "abion.test",
				},
			},
			expected: struct {
				keys   int
				values int
			}{
				keys:   1,
				values: 1,
			},
		},
		{
			name:     "zone and subdomain grouped",
			provider: AbionProvider{},
			zoneNameIDMapper: provider.ZoneIDName{
				"abion.test": "abion.test",
			},
			endpoints: []*endpoint.Endpoint{
				{
					DNSName: "abion.test",
				},
				{
					DNSName: "www.abion.test",
				},
			},
			expected: struct {
				keys   int
				values int
			}{
				keys:   1,
				values: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_populateZoneIdMapper(t *testing.T) {
	type testCase struct {
		name     string
		provider AbionProvider
		expected struct {
			dnsName string
			zoneId  string
			err     bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := tc.provider.populateZoneIdMapper(context.Background())
		checkError(t, err, tc.expected.err)
		if err == nil {
			id, _ := actual.FindZone(tc.expected.dnsName)
			assert.Equal(t, tc.expected.zoneId, id)
		}
	}

	testCases := []testCase{
		{
			name: "No records",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  0,
									Total:  0,
								},
							},
							Data: []internal.Zone{},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				dnsName string
				zoneId  string
				err     bool
			}{
				dnsName: "",
				zoneId:  "",
			},
		},
		{
			name: "Records returned",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  1,
									Total:  1,
								},
							},
							Data: []internal.Zone{
								{
									Type: "zone",
									ID:   "abion.test",
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				dnsName string
				zoneId  string
				err     bool
			}{
				dnsName: "abion.test",
				zoneId:  "abion.test",
			},
		},
		{
			name: "find zone id from subdomain",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{
									Offset: 0,
									Limit:  1,
									Total:  1,
								},
							},
							Data: []internal.Zone{
								{
									Type: "zone",
									ID:   "abion.test",
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				dnsName string
				zoneId  string
				err     bool
			}{
				dnsName: "www.abion.test",
				zoneId:  "abion.test",
			},
		},
		{
			name: "error fetching zones",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				dnsName string
				zoneId  string
				err     bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_processCreateActions(t *testing.T) {
	type testCase struct {
		name            string
		createsByDomain map[string][]*endpoint.Endpoint
		provider        AbionProvider
		expected        struct {
			err bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.provider.processCreateActions(context.Background(), tc.createsByDomain)
		checkError(t, err, tc.expected.err)
	}

	testCases := []testCase{
		{
			name:     "No records",
			provider: AbionProvider{},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "records created",
			createsByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"test.abion.test"},
						RecordType: "CNAME",
					},
					{
						DNSName:    "www.abion.test",
						Targets:    endpoint.Targets{"172.16.0.1"},
						RecordType: "A",
					},
				},
			},
			provider: AbionProvider{
				Client: &mockClient{
					patchZone: patchZoneResponse{
						APIResponse: nil,
						err:         nil,
					},
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"TXT": {
												{
													TTL:  3600,
													Data: "Existing TXT data",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "error patching zone",
			createsByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
				},
			},
			provider: AbionProvider{
				Client: &mockClient{
					patchZone: patchZoneResponse{
						APIResponse: nil,
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"TXT": {
												{
													TTL:  3600,
													Data: "Existing TXT data",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_processUpdateActions(t *testing.T) {
	type testCase struct {
		name               string
		updatesByDomain    map[string][]*endpoint.Endpoint
		updatesByDomainOld map[string][]*endpoint.Endpoint
		provider           AbionProvider
		expected           struct {
			err bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.provider.processUpdateActions(context.Background(), tc.updatesByDomain, tc.updatesByDomainOld)
		checkError(t, err, tc.expected.err)
	}

	testCases := []testCase{
		{
			name:     "No records",
			provider: AbionProvider{},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "error fetching zone",
			updatesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
				},
			},
			updatesByDomainOld: map[string][]*endpoint.Endpoint{},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "update zone",
			updatesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.1"},
						RecordType: "A",
					},
				},
			},
			updatesByDomainOld: map[string][]*endpoint.Endpoint{},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.0",
												},
											},
										},
										"www": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.1",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
					patchZone: patchZoneResponse{},
				},
			},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "error patching zone",
			updatesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.1"},
						RecordType: "A",
					},
				},
			},
			updatesByDomainOld: map[string][]*endpoint.Endpoint{},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.0",
												},
											},
										},
										"www": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.1",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
					patchZone: patchZoneResponse{
						APIResponse: nil,
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_processDeleteActions(t *testing.T) {
	type testCase struct {
		name            string
		deletesByDomain map[string][]*endpoint.Endpoint
		provider        AbionProvider
		expected        struct {
			err bool
		}
	}

	run := func(t *testing.T, tc testCase) {
		err := tc.provider.processDeleteActions(context.Background(), tc.deletesByDomain)
		checkError(t, err, tc.expected.err)
	}

	testCases := []testCase{
		{
			name:     "No records",
			provider: AbionProvider{},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "error fetching zone",
			deletesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
				},
			},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "delete zone record",
			deletesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
				},
			},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.0",
												},
											},
										},
										"www": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.1",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
					patchZone: patchZoneResponse{},
				},
			},
			expected: struct {
				err bool
			}{
				err: false,
			},
		},
		{
			name: "error patching zone",
			deletesByDomain: map[string][]*endpoint.Endpoint{
				"abion.test": {
					{
						DNSName:    "abion.test",
						Targets:    endpoint.Targets{"172.16.0.0"},
						RecordType: "A",
					},
				},
			},
			provider: AbionProvider{
				Client: &mockClient{
					getZone: zoneResponse{
						APIResponse: &internal.APIResponse[*internal.Zone]{
							Data: &internal.Zone{
								Type: "zone",
								ID:   "abion.test",
								Attributes: internal.Attributes{
									Records: map[string]map[string][]internal.Record{
										"@": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.0",
												},
											},
										},
										"www": {
											"A": {
												{
													TTL:  3600,
													Data: "172.16.0.1",
												},
											},
										},
									},
								},
							},
						},
						err: nil,
					},
					patchZone: patchZoneResponse{
						APIResponse: nil,
						err: &internal.Error{
							Status:  503,
							Message: "Service Unavailable",
						},
					},
				},
			},
			expected: struct {
				err bool
			}{
				err: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_formatTarget(t *testing.T) {
	type testCase struct {
		name     string
		provider AbionProvider
		target   string
		endpoint *endpoint.Endpoint
		expected struct {
			target string
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.provider.formatTarget(tc.endpoint, tc.target)
		assert.Equal(t, tc.expected.target, actual)
	}

	testCases := []testCase{
		{
			name:     "A record targets not changed",
			provider: AbionProvider{},
			endpoint: &endpoint.Endpoint{
				RecordType: "A",
			},
			target: "172.16.0.0",
			expected: struct {
				target string
			}{
				target: "172.16.0.0",
			},
		},
		{
			name:     "CNAME record target not changed if already suffixed",
			provider: AbionProvider{},
			endpoint: &endpoint.Endpoint{
				RecordType: "CNAME",
			},
			target: "abion.test.",
			expected: struct {
				target string
			}{
				target: "abion.test.",
			},
		},
		{
			name:     "CNAME record target suffixed",
			provider: AbionProvider{},
			endpoint: &endpoint.Endpoint{
				RecordType: "CNAME",
			},
			target: "abion.test",
			expected: struct {
				target string
			}{
				target: "abion.test.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_getAbionDnsName(t *testing.T) {
	type testCase struct {
		name     string
		provider AbionProvider
		zoneId   string
		endpoint *endpoint.Endpoint
		expected struct {
			name string
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.provider.getAbionDnsName(tc.endpoint.DNSName, tc.zoneId)
		assert.Equal(t, tc.expected.name, actual)
	}

	testCases := []testCase{
		{
			name:     "dns name same as zone id",
			provider: AbionProvider{},
			endpoint: &endpoint.Endpoint{
				DNSName: "abion.test",
			},
			zoneId: "abion.test",
			expected: struct {
				name string
			}{
				name: "@",
			},
		},
		{
			name:     "get sub domain",
			provider: AbionProvider{},
			endpoint: &endpoint.Endpoint{
				DNSName: "www.abion.test",
			},
			zoneId: "abion.test",
			expected: struct {
				name string
			}{
				name: "www",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func Test_NewAbionProvider_DomainFilterSplitting(t *testing.T) {
	type testCase struct {
		name                   string
		domainFilter           []string
		expectedZoneFilter     []string
		expectedDomainIncludes []string
		expectedDomainExcludes []string
	}

	testCases := []testCase{
		{
			name:                   "plain domains passed through unchanged",
			domainFilter:           []string{"example.com", "other.com"},
			expectedZoneFilter:     []string{"example.com", "other.com"},
			expectedDomainIncludes: []string{"example.com", "other.com"},
			expectedDomainExcludes: []string{},
		},
		{
			name:                   "wildcard converted to suffix for domainFilter",
			domainFilter:           []string{"*.example.com"},
			expectedZoneFilter:     []string{"*.example.com"},
			expectedDomainIncludes: []string{"sub.example.com", "deep.sub.example.com"},
			expectedDomainExcludes: []string{"example.com", "other.com"},
		},
		{
			name:                   "mixed plain and wildcard",
			domainFilter:           []string{"exact.com", "*.wild.com"},
			expectedZoneFilter:     []string{"exact.com", "*.wild.com"},
			expectedDomainIncludes: []string{"exact.com", "sub.wild.com"},
			expectedDomainExcludes: []string{"wild.com", "other.com"},
		},
		{
			name:                   "whitespace trimmed and empty entries skipped",
			domainFilter:           []string{"  example.com  ", "", "  "},
			expectedZoneFilter:     []string{"example.com"},
			expectedDomainIncludes: []string{"example.com"},
			expectedDomainExcludes: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &configuration.Configuration{
				ApiKey:       "test-key",
				DomainFilter: tc.domainFilter,
			}
			p, err := NewAbionProvider(config)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedZoneFilter, p.zoneFilter)

			for _, domain := range tc.expectedDomainIncludes {
				assert.True(t, p.domainFilter.Match(domain), "domainFilter should match %s", domain)
			}
			for _, domain := range tc.expectedDomainExcludes {
				assert.False(t, p.domainFilter.Match(domain), "domainFilter should not match %s", domain)
			}
		})
	}
}

func Test_hasWildcardFilter(t *testing.T) {
	tests := []struct {
		name       string
		zoneFilter []string
		expected   bool
	}{
		{"no wildcard", []string{"example.com", "other.com"}, false},
		{"has wildcard", []string{"example.com", "*.wild.com"}, true},
		{"only wildcard", []string{"*.example.com"}, true},
		{"empty filter", []string{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &AbionProvider{zoneFilter: tc.zoneFilter}
			assert.Equal(t, tc.expected, p.hasWildcardFilter())
		})
	}
}

func Test_matchesZoneFilter(t *testing.T) {
	tests := []struct {
		name       string
		zoneFilter []string
		zone       string
		expected   bool
	}{
		{"exact match", []string{"example.com"}, "example.com", true},
		{"no match", []string{"example.com"}, "other.com", false},
		{"wildcard matches subdomain", []string{"*.example.com"}, "sub.example.com", true},
		{"wildcard matches deep subdomain", []string{"*.example.com"}, "deep.sub.example.com", true},
		{"wildcard does not match root", []string{"*.example.com"}, "example.com", false},
		{"mixed filter exact match", []string{"exact.com", "*.wild.com"}, "exact.com", true},
		{"mixed filter wildcard match", []string{"exact.com", "*.wild.com"}, "sub.wild.com", true},
		{"mixed filter no match", []string{"exact.com", "*.wild.com"}, "other.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &AbionProvider{zoneFilter: tc.zoneFilter}
			assert.Equal(t, tc.expected, p.matchesZoneFilter(tc.zone))
		})
	}
}

func Test_getFilteredZoneIDs(t *testing.T) {
	type testCase struct {
		name     string
		provider AbionProvider
		expected struct {
			zoneIDs []string
			err     bool
		}
	}

	testCases := []testCase{
		{
			name: "no filter returns all zones",
			provider: AbionProvider{
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{Offset: 0, Limit: 2, Total: 2},
							},
							Data: []internal.Zone{
								{ID: "a.com"}, {ID: "b.com"},
							},
						},
					},
				},
			},
			expected: struct {
				zoneIDs []string
				err     bool
			}{zoneIDs: []string{"a.com", "b.com"}},
		},
		{
			name: "plain filter returns filter directly without API call",
			provider: AbionProvider{
				zoneFilter: []string{"specific.com"},
				Client:     &mockClient{}, // no getZones configured - would panic if called incorrectly
			},
			expected: struct {
				zoneIDs []string
				err     bool
			}{zoneIDs: []string{"specific.com"}},
		},
		{
			name: "wildcard filter fetches all zones and matches",
			provider: AbionProvider{
				zoneFilter: []string{"*.example.com"},
				Client: &mockClient{
					getZones: zonesResponse{
						APIResponse: &internal.APIResponse[[]internal.Zone]{
							Meta: &internal.Metadata{
								Pagination: &internal.Pagination{Offset: 0, Limit: 3, Total: 3},
							},
							Data: []internal.Zone{
								{ID: "sub.example.com"},
								{ID: "other.com"},
								{ID: "deep.sub.example.com"},
							},
						},
					},
				},
			},
			expected: struct {
				zoneIDs []string
				err     bool
			}{zoneIDs: []string{"sub.example.com", "deep.sub.example.com"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.provider.getFilteredZoneIDs(context.Background())
			checkError(t, err, tc.expected.err)
			if err == nil {
				assert.Equal(t, tc.expected.zoneIDs, actual)
			}
		})
	}
}

func Test_getExternalDnsDnsName(t *testing.T) {
	type testCase struct {
		name         string
		provider     AbionProvider
		zoneId       string
		abionDnsName string
		expected     struct {
			name string
		}
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.provider.getExternalDnsDnsName(tc.abionDnsName, tc.zoneId)
		assert.Equal(t, tc.expected.name, actual)
	}

	testCases := []testCase{
		{
			name:         "from root abion dns name",
			provider:     AbionProvider{},
			abionDnsName: "@",
			zoneId:       "abion.test",
			expected: struct {
				name string
			}{
				name: "abion.test",
			},
		},
		{
			name:         "from sub abion dns name",
			provider:     AbionProvider{},
			abionDnsName: "www",
			zoneId:       "abion.test",
			expected: struct {
				name string
			}{
				name: "www.abion.test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
