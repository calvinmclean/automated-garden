//
//  EditZoneHost.swift
//  Garden App
//
//  Created by Calvin McLean on 12/26/22.
//

import SwiftUI

struct EditZoneHost: View {
    @EnvironmentObject var modelData: ModelData
    @State private var newZone: UpdateZoneRequest
    var garden: Garden
    var zone: Zone
    
    init(garden: Garden, zone: Zone) {
        self.garden = garden
        self.zone = zone
        self.newZone = UpdateZoneRequest(position: zone.position)
    }
    
    var body: some View {
        EditZoneEditor(garden: garden, zone: zone, newZone: $newZone)
            .onAppear {
                newZone = UpdateZoneRequest(position: zone.position)
            }
//            .onDisappear {
//                modelData.fetchZones(garden: garden)
//            }
    }
}
