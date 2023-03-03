//
//  AddPlantSheet.swift
//  Garden App
//
//  Created by Calvin McLean on 6/28/21.
//

import SwiftUI

struct AddZoneHost: View {
    @EnvironmentObject var modelData: ModelData
    @State private var newZone: CreateZoneRequest = CreateZoneRequest()
    var garden: Garden
    
    init(garden: Garden) {
        self.garden = garden
    }
    
    var body: some View {
        AddZoneEditor(garden: garden, newZone: $newZone)
            .onAppear {
                newZone = CreateZoneRequest()
            }
            .onDisappear {
                modelData.fetchZones(garden: garden)
            }
    }
}
