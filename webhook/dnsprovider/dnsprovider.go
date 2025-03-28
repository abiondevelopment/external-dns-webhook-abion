package dnsprovider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/abiondevelopment/external-dns-webhook-abion/internal"
	"github.com/abiondevelopment/external-dns-webhook-abion/webhook/configuration"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

type AbionProvider struct {
	provider.BaseProvider
	Client internal.ApiClient
	DryRun bool
}

func NewAbionProvider(config *configuration.Configuration) (*AbionProvider, error) {
	client := *internal.NewAbionClient(config.ApiKey)
	p := &AbionProvider{
		Client: &client,
		DryRun: config.DryRun,
	}

	return p, nil
}

// Records returns the list of records for all zones your session can access
func (p *AbionProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	var endpoints []*endpoint.Endpoint

	offset := 0
	for {
		page := &internal.Pagination{
			Offset: offset,
		}

		zonesResponse, err := p.Client.GetZones(ctx, page)
		if err != nil {
			return nil, err
		}

		for _, zoneData := range zonesResponse.Data {
			zone, err := p.Client.GetZone(ctx, zoneData.ID)
			if err != nil {
				return nil, err
			}

			for dnsName, record := range zone.Data.Attributes.Records {
				// recordDetails map[string][]internal.Record
				for recordType, recordDetails := range record {
					// recordType string A, AAAA, CAA, CNAME, DNAME, LOC, MX, NAPTR, NS, PTR, RP, SRV, SSHFP, TLSA, TXT
					// recordDetails []Record
					for _, recordDetail := range recordDetails {
						ep := endpoint.NewEndpointWithTTL(p.getExternalDnsDnsName(dnsName, zoneData.ID), recordType, endpoint.TTL(recordDetail.TTL), recordDetail.Data)
						endpoints = append(endpoints, ep)
					}
				}
			}
		}

		offset = page.Offset + len(zonesResponse.Data)

		if offset >= zonesResponse.Meta.Total {
			break
		}
	}

	log.WithFields(log.Fields{
		"endpoints": endpoints,
	}).Debug("Records")

	return endpoints, nil
}

func (p *AbionProvider) endpointsByZone(zoneNameIDMapper provider.ZoneIDName, endpoints []*endpoint.Endpoint) map[string][]*endpoint.Endpoint {
	endpointsByZone := make(map[string][]*endpoint.Endpoint)

	for _, ep := range endpoints {
		zoneID, _ := zoneNameIDMapper.FindZone(ep.DNSName)
		if zoneID == "" {
			log.Debugf("Skipping record %s because no hosted zone matching record DNS Name was detected", ep.DNSName)
			continue
		}
		endpointsByZone[zoneID] = append(endpointsByZone[zoneID], ep)
	}

	return endpointsByZone
}

// ApplyChanges applies a given set of changes for zones
func (p *AbionProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	zoneNameIDMapper, err := p.populateZoneIdMapper(ctx)
	if err != nil {
		return err
	}

	createsByDomain := p.endpointsByZone(zoneNameIDMapper, changes.Create)
	updatesByDomainNew := p.endpointsByZone(zoneNameIDMapper, changes.UpdateNew)
	updatesByDomainOld := p.endpointsByZone(zoneNameIDMapper, changes.UpdateOld)
	deletesByDomain := p.endpointsByZone(zoneNameIDMapper, changes.Delete)

	if err := p.processCreateActions(ctx, createsByDomain); err != nil {
		return err
	}

	if err := p.processUpdateActions(ctx, updatesByDomainNew, updatesByDomainOld); err != nil {
		return err
	}

	if err := p.processDeleteActions(deletesByDomain); err != nil {
		return err
	}
	return nil
}

func (p *AbionProvider) populateZoneIdMapper(ctx context.Context) (provider.ZoneIDName, error) {
	var zoneIds []string
	offset := 0
	for {
		page := &internal.Pagination{
			Offset: offset,
		}
		zonesResponse, err := p.Client.GetZones(ctx, page)
		if err != nil {
			return nil, err
		}

		for _, data := range zonesResponse.Data {
			zoneIds = append(zoneIds, data.ID)
		}
		offset = page.Offset + len(zonesResponse.Data)
		if offset >= zonesResponse.Meta.Total {
			break
		}
	}

	// populate zoneIDMapper
	zoneNameIDMapper := provider.ZoneIDName{}
	for _, zoneId := range zoneIds {
		zoneNameIDMapper.Add(zoneId, zoneId)
	}
	return zoneNameIDMapper, nil
}

func (p *AbionProvider) processCreateActions(ctx context.Context, createsByDomain map[string][]*endpoint.Endpoint) error {
	for zoneId, createEndpoints := range createsByDomain {

		zone, err := p.Client.GetZone(ctx, zoneId)
		if err != nil {
			return err
		}

		records := make(map[string]map[string][]internal.Record)

		for _, createEndpoint := range createEndpoints {

			dnsName := p.getAbionDnsName(createEndpoint.DNSName, zoneId)

			var data []internal.Record
			for _, target := range createEndpoint.Targets {
				target = p.formatTarget(createEndpoint, target)

				var record internal.Record
				record = p.createRecord(createEndpoint, record, target)
				data = append(data, record)
			}

			if records[dnsName] == nil {
				records[dnsName] = make(map[string][]internal.Record)
			}

			// add all existing records to make sure to not clear the other zone records on same name level and of same record type
			log.WithFields(
				log.Fields{
					"dnsName": dnsName,
				}).Debug("Checking existing zone records on name level")
			existingRecordsByDNSName, ok := zone.Data.Attributes.Records[dnsName]
			if ok {
				log.WithFields(
					log.Fields{
						"dnsName":    dnsName,
						"recordType": createEndpoint.RecordType,
					}).Debug("Checking existing zone records of same record type on same dns name level")
				existingRecordsOfSameRecordType, ok := existingRecordsByDNSName[createEndpoint.RecordType]
				if ok {
					data = append(data, existingRecordsOfSameRecordType...)
				}
			}

			records[dnsName][createEndpoint.RecordType] = data
		}

		log.WithFields(log.Fields{
			"records": records,
		}).Debug("Create records")

		if p.DryRun {
			continue
		}

		err = p.submitPatchZone(zoneId, records)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *AbionProvider) processUpdateActions(ctx context.Context, updatesByDomainNew map[string][]*endpoint.Endpoint, updatesByDomainOld map[string][]*endpoint.Endpoint) error {
	for zoneId, updateEndpointsNew := range updatesByDomainNew {

		currentZone, err := p.Client.GetZone(ctx, zoneId)
		if err != nil {
			log.Warnf("unable to get zone: %s, error: %v", zoneId, err)
			continue
		}

		records := make(map[string]map[string][]internal.Record)

		for _, updateEndpointNew := range updateEndpointsNew {

			dnsName := p.getAbionDnsName(updateEndpointNew.DNSName, zoneId)

			var data []internal.Record
			currentSubDomain, subdomainExist := currentZone.Data.Attributes.Records[dnsName] // subdomain www or @ if root
			if subdomainExist {
				currentRecords, recordsExist := currentSubDomain[updateEndpointNew.RecordType] // current records of type TXT, A, etc., on same dns name/subdomain (@, www, etc)
				if recordsExist {
					for _, target := range updateEndpointNew.Targets {
						// always add the updated (new) changes from external-dns
						target = p.formatTarget(updateEndpointNew, target)
						log.WithFields(
							log.Fields{
								"recordType": updateEndpointNew.RecordType,
								"target":     target,
							}).Debug("Adding updated (new) zone record")
						var record internal.Record
						record = p.createRecord(updateEndpointNew, record, target)
						data = append(data, record)
					}
					for _, currentRecord := range currentRecords {
						// need to add all the current records which are not included in update (old) changes
						addRecord := true
						for _, updateEndpointsOld := range updatesByDomainOld {
							for _, updateEndpointOld := range updateEndpointsOld {
								if updateEndpointOld.RecordType == updateEndpointNew.RecordType { // make sure compare same record type
									for _, oldTarget := range updateEndpointOld.Targets {
										oldTarget = p.formatTarget(updateEndpointOld, oldTarget)
										if currentRecord.Data == oldTarget { // if any old target matches with current record data, it means it has already been added by the updated (new) changes from external-dns.
											log.WithFields(
												log.Fields{
													"currentRecord": currentRecord,
												}).Debug("Don't add this record as an updated version has already been added by the updated (new) changes from external-dns")
											addRecord = false
										}
									}
								}
							}
						}
						if addRecord {
							log.WithFields(
								log.Fields{
									"currentRecord": currentRecord,
								}).Debug("Adding current zone record")
							data = append(data, currentRecord)
						}
					}
				}
			}

			if records[dnsName] == nil {
				records[dnsName] = make(map[string][]internal.Record)
			}
			records[dnsName][updateEndpointNew.RecordType] = data
		}

		log.WithFields(log.Fields{
			"records": records,
		}).Debug("Update records")

		if p.DryRun {
			continue
		}

		err = p.submitPatchZone(zoneId, records)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *AbionProvider) processDeleteActions(deletesByDomain map[string][]*endpoint.Endpoint) error {
	for zoneId, deleteEndpoints := range deletesByDomain {

		currentZone, err := p.Client.GetZone(context.Background(), zoneId)
		if err != nil {
			log.Warnf("unable to get zone: %s, error: %v", zoneId, err)
			continue
		}

		records := make(map[string]map[string][]internal.Record)

		for _, deleteEndpoint := range deleteEndpoints {

			dnsName := p.getAbionDnsName(deleteEndpoint.DNSName, zoneId)

			var data []internal.Record
			currentSubDomain, subDomainExist := currentZone.Data.Attributes.Records[dnsName] // subdomain www or @ if root
			if subDomainExist {
				existingRecordsForRecordType, recordsExist := currentSubDomain[deleteEndpoint.RecordType]
				if recordsExist {
					var targets []string
					for _, target := range deleteEndpoint.Targets {
						target = p.formatTarget(deleteEndpoint, target)
						targets = append(targets, target)
					}

					remainingRecords := slices.DeleteFunc(existingRecordsForRecordType, func(r internal.Record) bool {
						return slices.Contains(targets, r.Data)
					})
					data = remainingRecords
				}
			}

			if records[dnsName] == nil {
				records[dnsName] = make(map[string][]internal.Record)
			}
			records[dnsName][deleteEndpoint.RecordType] = data
		}
		log.WithFields(log.Fields{
			"records": records,
		}).Debug("Delete records")

		if p.DryRun {
			continue
		}

		err = p.submitPatchZone(zoneId, records)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *AbionProvider) submitPatchZone(zoneId string, records map[string]map[string][]internal.Record) error {
	patchRequest := internal.ZoneRequest{
		Data: internal.Zone{
			Type: "zone",
			ID:   zoneId,
			Attributes: internal.Attributes{
				Records: records,
			},
		},
	}

	_, err := p.Client.PatchZone(context.Background(), zoneId, patchRequest)
	if err != nil {
		return fmt.Errorf("error updating zone %w", err)
	}

	return nil
}

func (p *AbionProvider) formatTarget(endpoint *endpoint.Endpoint, target string) string {
	if endpoint.RecordType == "CNAME" && !strings.HasSuffix(target, ".") {
		target += "."
	}
	return target
}

func (p *AbionProvider) createRecord(createEndpoint *endpoint.Endpoint, record internal.Record, target string) internal.Record {
	if createEndpoint.RecordTTL.IsConfigured() {
		record = internal.Record{
			Data: target,
			TTL:  int(createEndpoint.RecordTTL),
		}
	} else {
		record = internal.Record{
			Data: target,
		}
	}
	return record
}

func (p *AbionProvider) getAbionDnsName(dnsName string, zoneId string) string {
	// adjust name to @ or subDomain, e.g. www
	if zoneId == dnsName {
		return "@"
	} else {
		return strings.TrimSuffix(dnsName, "."+zoneId) // www.abion.com -> www
	}
}

func (p *AbionProvider) getExternalDnsDnsName(dnsName string, zoneId string) string {
	if dnsName == "@" {
		return zoneId
	} else {
		return dnsName + "." + zoneId
	}
}
