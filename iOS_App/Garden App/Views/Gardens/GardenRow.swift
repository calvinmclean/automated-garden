//
//  GardenRow.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/21.
//

import SwiftUI

struct GardenRow: View {
    @EnvironmentObject var modelData: ModelData
    @State private var isShowingDeleteConfirmation = false
    @State var nextLightTimeState: String = ""
    let garden: Garden
    
    var createdAtDate: String {
        if let endDate = garden.endDate {
            return "\(garden.createdAt.formatted) - \(endDate.formatted)"
        }
        return garden.createdAt.formatted
    }
    
    let timer = Timer.publish(
        every: 5, // second
        on: .main,
        in: .common
    ).autoconnect()
    
    var body: some View {
        let stopWateringButton = Button(
            action: {
                modelData.gardenResource().stopWatering(garden: garden)
            },
            label: {
                Label { Text("Stop Watering") } icon: { Image(systemName: "exclamationmark.octagon") }
            }
        )
            .disabled(garden.isEndDated())
        
        let toggleLightButton = Button(
            action: {
                modelData.gardenResource().toggleLight(garden: garden)
            },
            label: {
                Label { Text("Toggle Light") } icon: { Image(systemName: "sun.max") }
            }
        )
            .disabled(garden.isEndDated())
            .disabled(garden.lightSchedule == nil)
        
        let deleteButton = Button(
            role: .destructive,
            action: {
                if garden.isEndDated() {
                    isShowingDeleteConfirmation = true
                } else {
                    modelData.gardenResource().endDateGarden(garden: garden)
                    modelData.fetchGardens()
                }
            },
            label: {
                Label { Text("Delete") } icon: { Image(systemName: "trash") }
            }
        )
        
        let restoreButton = Button(
            action: {
                modelData.gardenResource().restoreGarden(garden: garden)
                modelData.fetchGardens()
            },
            label: {
                Label { Text("Restore") } icon: { Image(systemName: "arrow.uturn.backward") }
            }
        )
            .disabled(!garden.isEndDated())
        
        VStack(alignment: .leading, spacing: 8.0) {
            Label { Text(garden.name) } icon: { Image(systemName: "square.grid.2x2")
                .foregroundColor(garden.isEndDated() ? .red : .green)}.font(.headline)
            HStack {
                Text(createdAtDate)
                    .font(.caption)
                    .foregroundColor(.secondary)
                Divider()
                Text(garden.id)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if !garden.isEndDated() {
                HStack {
                    if let up = garden.health?.status == "UP" {
                        Label(garden.health?.status ?? "N/A", systemImage: up ? "wifi" : "wifi.slash").foregroundColor(up ? .blue : .red)
                    }
                    Label(String(garden.numZones), systemImage: "drop.triangle").foregroundColor(.yellow)
                    Label(String(garden.numPlants), systemImage: "leaf.circle").foregroundColor(.green)
                    if let nextLightAction = garden.nextLightAction {
                        Label(nextLightTimeState, systemImage: nextLightAction.state == "ON" ? "sunrise" : "sunset")
                            .foregroundColor(.yellow)
                            .onReceive(timer) { (_) in
                                // if this is in the past, do not update view (TODO: actually fetch the Garden's new time)
                                if nextLightAction.time.timeIntervalSinceNow > 0 {
                                    nextLightTimeState = nextLightAction.time.minFormatted
                                }
                            }
                            .onAppear() {
                                nextLightTimeState = nextLightAction.time.minFormatted
                            }
                    }
                }
            }
        }
        .padding(.top, 24.0)
        .padding(.bottom, 16.0)
        .contextMenu(menuItems: {
            stopWateringButton
            toggleLightButton
            deleteButton
            
            Divider()
            
            restoreButton
        })
        .swipeActions(edge: .leading) {
            stopWateringButton.tint(.red)
            toggleLightButton.tint(.yellow)
            restoreButton.tint(.gray)
        }
        .swipeActions(edge: .trailing) {
            deleteButton.tint(.red)
        }
        .confirmationDialog("Are you sure you want to permanently delete the data?", isPresented: $isShowingDeleteConfirmation, titleVisibility: .visible) {
            Button("Confirm", role: .destructive) {
                modelData.gardenResource().endDateGarden(garden: garden)
                modelData.fetchGardens()
            }
        }
    }
}

struct GardenRow_Preview: PreviewProvider {
    static var previews: some View {
        let garden: Garden = {
            let garden = Garden()
            garden.name = "My Garden"
            garden.numZones = 3
            garden.numPlants = 8
            garden.id = "garden_id"
            garden.health = GardenHealth()
            garden.health?.status = "UP"
            garden.nextLightAction = NextLightAction()
            garden.nextLightAction?.time = Date().advanced(by: 60 * 60 * 48)
            return garden
        }()
        List {
            GardenRow(garden: garden)
        }
    }
}
