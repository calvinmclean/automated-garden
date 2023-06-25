//
//  GardenDetail.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/21.
//

import Foundation
import SwiftUI

struct GardenDetail: View {
    @EnvironmentObject var modelData: ModelData
    @State private var sortBy = Zone.SortBy.position
    @State private var reverseSort = false
    @State private var showingAddZone = false
    
    let garden: Garden
    
    func toggleSortBy(buttonPressed: Zone.SortBy) {
        switch sortBy {
        case buttonPressed: // If buttonPressed is the same as current selection, reset
            sortBy = Zone.SortBy.position
        default:
            sortBy = buttonPressed
        }
    }
    
    var filteredZones: [Zone] {
        let sortedZones = modelData.zones
            .filter { !$0.isEndDated() }
            .sorted {
                $0.isLessThan(other: $1, sortBy: sortBy)
            }
        return reverseSort ? sortedZones.reversed() : sortedZones
    }
    
    var endDatedZones: [Zone] {
        let sortedZones = modelData.zones
            .filter { $0.isEndDated() }
            .sorted {
                $0.isLessThan(other: $1, sortBy: sortBy)
            }
        return reverseSort ? sortedZones.reversed() : sortedZones
    }
    
    func endDateZone(at indexSet: IndexSet) {
        for index in indexSet {
            modelData.zoneResource().endDateZone(zone: filteredZones[index])
            modelData.fetchZones(garden: garden)
        }
    }
    
    func permanentDeleteZone(at indexSet: IndexSet) {
        for index in indexSet {
            modelData.zoneResource().endDateZone(zone: endDatedZones[index])
            modelData.fetchZones(garden: garden)
        }
    }
    
    var body: some View {
        NavigationView {
            List {
                Section("Garden Actions") {
                    if !garden.isEndDated() {
                        NavigationLink(destination: GardenActionDetail(garden: garden)) {
                            Spacer()
                            
                            Button(action: {
                                print("Stop watering button tapped for \(garden.name)")
                                modelData.gardenResource().stopWatering(garden: garden)
                            }) {
                                Label { } icon: { Image(systemName: "exclamationmark.octagon.fill") }
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .red))
                            .controlSize(.large)
                            
                            Spacer()
                            
                            if (garden.lightSchedule != nil) {
                                Button(action: {
                                    print("Toggle light button tapped for \(garden.name)")
                                    modelData.gardenResource().toggleLight(garden: garden)
                                }) {
                                    Label { } icon: { Image(systemName: "sun.max") }
                                }
                                .buttonStyle(ActionButtonStyle(bgColor: .yellow))
                                .controlSize(.large)
                                
                                Spacer()
                                
                                Button(action: {
                                    print("Delay light button tapped for \(garden.name)")
                                    modelData.gardenResource().delayLight(garden: garden)
                                }) {
                                    Label { } icon: { Image(systemName: "cloud.sun") }
                                }
                                .buttonStyle(ActionButtonStyle(bgColor: .gray))
                                .controlSize(.large)
                                
                                Spacer()
                            }
                        }
                    }
                }
                Section("Active Zones") {
                    ForEach(filteredZones) { zone in
                        NavigationLink(destination: ZoneDetail(zone: zone, garden: garden)) {
                            ZoneRow(garden: garden, zone: zone)
                        }
                    }
                    .onDelete(perform: self.endDateZone)
                }
                Section("End Dated Zones") {
                    ForEach(endDatedZones) { zone in
                        NavigationLink(destination: ZoneDetail(zone: zone, garden: garden)) {
                            ZoneRow(garden: garden, zone: zone)
                        }
                    }
                    .onDelete(perform: self.permanentDeleteZone)
                }
            }
            .refreshable{ modelData.fetchZones(garden: garden) }
            .listStyle(InsetGroupedListStyle())
            .onAppear { modelData.fetchZones(garden: garden) }
            .toolbar {
                ToolbarItemGroup(placement: .bottomBar) {
                    Button(action: { toggleSortBy(buttonPressed: .position) }) {
                        Image(systemName: (sortBy == .position ? "1.circle.fill" : "1.circle"))
                            .accessibilityLabel("Toggle Sort By Zone Position")
                    }
                    Button(action: { toggleSortBy(buttonPressed: .name) }) {
                        Image(systemName: (sortBy == .name ? "a.circle.fill" : "a.circle"))
                            .accessibilityLabel("Toggle Sort By Name")
                    }
                    Button(action: { toggleSortBy(buttonPressed: .createdAt) }) {
                        Image(systemName: (sortBy == .createdAt ? "calendar.circle.fill" : "calendar.circle"))
                            .accessibilityLabel("Toggle Sort By Start Date")
                    }
//                    Button(action: { toggleSortBy(buttonPressed: .nextWatering) }) {
//                        Image(systemName: (sortBy == .nextWatering ? "drop.fill" : "drop"))
//                            .accessibilityLabel("Toggle Sort By Next Watering Time")
//                    }
//                    Divider()
                    Button(action: { reverseSort.toggle() }) {
                        Image(systemName: (reverseSort ? "arrow.up" : "arrow.down"))
                            .accessibilityLabel("Reverse Sort")
                    }
                    Spacer()
                    Button(action: { showingAddZone.toggle() }) {
                        Image(systemName: "plus")
                            .accessibilityLabel("Add Zone")
                    }
                }
            }
            .sheet(isPresented: $showingAddZone) {
                AddZoneHost(garden: garden)
                    .environmentObject(modelData)
                    .onDisappear { modelData.fetchZones(garden: garden) }
            }
        }
        .navigationTitle(garden.name)
    }
}
