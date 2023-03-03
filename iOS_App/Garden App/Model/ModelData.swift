//
//  ModelData.swift
//  Garden App
//
//  Created by Calvin McLean on 6/12/21.
//

import Foundation
import Combine

final class ModelData: ObservableObject {
    @Published var settings = Settings.default
    @Published var zones: [Zone] = []
    @Published var gardens: [Garden] = []
    @Published private(set) var isLoading = false
    @Published private(set) var isLoadingGarden = false
    
    func zoneResource() -> ZoneResource {
        ZoneResource(url: self.settings.server.url)
    }
    
    func gardenResource() -> GardenResource {
        GardenResource(url: self.settings.server.url)
    }
    
    func fetchGardens() {
        guard !isLoadingGarden else { return }
        isLoadingGarden = true
        gardenResource().getGardens(showEndDated: true) { [weak self] gardens in
            self?.gardens = gardens ?? []
        }
    }
    
    func fetchZones(garden: Garden) {
        guard !isLoading else { return }
        isLoading = true
        zoneResource().getZones(garden: garden, showEndDated: true) { [weak self] zones in
            self?.zones = zones ?? []
            self?.isLoading = false
        }
    }
    
    func fetchZoneWaterHistory(zone: Zone, range: String, limit: Int) {
        guard !isLoading else { return }
        isLoading = true
        zoneResource().getWateringHistory(zone: zone, range: range, limit: limit) { [weak self] history in
            zone.history = history ?? WaterHistoryResponse()
            self?.isLoading = false
        }
    }
}
