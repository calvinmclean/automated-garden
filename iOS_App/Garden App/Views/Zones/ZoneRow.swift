//
//  ZoneRow.swift
//  Garden App
//
//  Created by Calvin McLean on 6/12/21.
//

import SwiftUI

struct ZoneRow: View {
    @EnvironmentObject var modelData: ModelData
    @State private var isShowingDeleteConfirmation = false
    @State var nextWaterTimeState: String = ""
    let garden: Garden
    let zone: Zone
    
    var createdAtDate: String {
        if let endDate = zone.endDate {
            return "\(zone.createdAt.formatted) - \(endDate.formatted)"
        }
        return zone.createdAt.formatted
    }
    
    var body: some View {
        let quickWaterButton = Button(
            action: {
                modelData.zoneResource().waterZone(
                    zone: zone,
                    duration: 5000,
                    ignoreMoisture: true
                )
            },
            label: {
                Label { Text("Quick Water (5s)") } icon: { Image(systemName: "drop") }
            }
        )
            .disabled(zone.isEndDated())
        
        let quickDelayButton = Button(
            action: {
                modelData.zoneResource().delayWatering(zone: zone, days: 1)
            },
            label: {
                Label { Text("Quick Delay (+1 day)") } icon: { Image(systemName: "goforward") }
            }
        )
            .disabled(zone.isEndDated())
        
        let deleteButton = Button(
            role: .destructive,
            action: {
                if zone.isEndDated() {
                    isShowingDeleteConfirmation = true
                } else {
                    modelData.zoneResource().endDateZone(zone: zone)
                    modelData.fetchZones(garden: garden)
                }
            },
            label: {
                Label { Text("Delete") } icon: { Image(systemName: "trash") }
            }
        )
        
        let restoreButton = Button(
            action: {
                modelData.zoneResource().restoreZone(zone: zone)
                modelData.fetchZones(garden: garden)
            },
            label: {
                Label { Text("Restore") } icon: { Image(systemName: "arrow.uturn.backward") }
            }
        )
            .disabled(!zone.isEndDated())
        
        let timer = Timer.publish(
            every: 5, // second
            on: .main,
            in: .common
        ).autoconnect()
        
        VStack(alignment: .leading, spacing: 8.0) {
            Label { Text(zone.name) } icon: { Image(systemName: "leaf.fill")
                .foregroundColor(zone.isEndDated() ? .red : .green)}
            .font(.headline)
            HStack {
                Text(createdAtDate)
                    .font(.caption)
                    .foregroundColor(.secondary)
                Divider()
                Text(String(zone.id))
                    .font(.caption)
                    .foregroundColor(.secondary)
                Divider()
                Text(String(zone.position))
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if !zone.isEndDated() {
                HStack {
                    if let nextWaterTime = zone.nextWaterTime {
                        Label(nextWaterTimeState, systemImage: "drop")
                            .foregroundColor(.blue)
                            .onReceive(timer) { (_) in
                                // if this is in the past, do not update view (TODO: actually fetch the Zone's new time)
                                if nextWaterTime.timeIntervalSinceNow > 0 {
                                    nextWaterTimeState = nextWaterTime.minFormatted
                                }
                            }
                            .onAppear() {
                                nextWaterTimeState = nextWaterTime.minFormatted
                            }
                    }
                    Spacer()
                    if let moisture = zone.moisture {
                        Label(String(format: "%.f%%", moisture.rounded()), systemImage: "humidity")
                            .foregroundColor(.cyan)
                    }
                }
            }
        }
        .swipeActions(edge: .leading) {
            quickWaterButton.tint(.blue)
            quickDelayButton.tint(.orange)
            restoreButton.tint(.gray)
        }
        .swipeActions(edge: .trailing) {
            deleteButton.tint(.red)
        }
        .confirmationDialog("Are you sure you want to permanently delete the data?", isPresented: $isShowingDeleteConfirmation, titleVisibility: .visible) {
            Button("Confirm", role: .destructive) {
                modelData.zoneResource().endDateZone(zone: zone)
                modelData.fetchZones(garden: garden)
            }
        }
        .padding(.top, 24.0)
        .padding(.bottom, 16.0)
        .contextMenu(menuItems: {
            quickWaterButton
            quickDelayButton
            deleteButton
            
            Divider()
            
            restoreButton
        })
    }
}
