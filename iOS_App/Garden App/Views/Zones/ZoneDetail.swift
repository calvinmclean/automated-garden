//
//  ZoneDetail.swift
//  Garden App
//
//  Created by Calvin McLean on 6/11/21.
//

import SwiftUI

struct DetailHStack: View {
    var key: String
    var value: String?
    
    var body: some View {
        if let value = value {
            HStack {
                Text(key).frame(width: 100, alignment: .leading)
                Divider()
                Text(value)
            }
        }
    }
}

struct ZoneDetail: View {
    @EnvironmentObject var modelData: ModelData
    var zone: Zone
    var garden: Garden
    
    @State var wateringAmount: Int = 5
    @State var skipCount: Int = 0
    @State private var showingEditZone = false
    
    var body: some View {
        List {
            Section("Zone Actions") {
                if (!zone.isEndDated()) {
                    VStack {
                        Stepper(value: $wateringAmount, in: 5...300, step: 5) {
                            Button(action: {
                                print("Water Zone button tapped for \(zone.name)")
                                modelData.zoneResource().waterZone(
                                    zone: zone,
                                    duration: "\(wateringAmount)s"
                                )
                            }) {
                                Label { Text("\(wateringAmount) seconds") } icon: { Image(systemName: "cloud.rain.fill") }
                                    .frame(minWidth: 150, maxWidth: 150)
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .blue))
                            .controlSize(.large)
                        }
                    }.padding(10)
                    
                    VStack(alignment: .leading) {
                        Stepper(value: $skipCount, in: 0...100, step: 1) {
                            Button(action: {
                                print("Delay watering button tapped for \(zone.name)")
                                modelData.zoneResource().delayWatering(zone: zone, numSkip: skipCount)
                            }) {
                                Label { Text("Delay \(skipCount) day(s)") } icon: { Image(systemName: "goforward") }
                                    .frame(minWidth: 150, maxWidth: 150)
                            }
                            .onAppear {
                                skipCount = zone.skipCount ?? 0
                            }
                            .buttonStyle(ActionButtonStyle(bgColor: .orange))
                            .controlSize(.large)
                        }
                    }.padding(10)
                }
            }
            
            if let details = zone.details {
                Section("Details") {
                    if let description = details.description {
                        DetailHStack(key: "Description", value: description)
                    }
                    if let notes = details.notes {
                        DetailHStack(key: "Notes", value: notes)
                    }
                    DetailHStack(key: "Created At", value: zone.createdAt.formatted)
                }
            }
            
            Section("Water Schedule") {
                if let nextWaterDetails = zone.nextWaterDetails {
                    if let nextWateringTime = nextWaterDetails.time {
                        DetailHStack(key: "Next Watering Time", value: nextWateringTime.formattedWithTime)
                    }
                    if let nextWateringDuration = nextWaterDetails.duration {
                        DetailHStack(key: "Next Watering Duration", value: nextWateringDuration)
                    }
                    if let nextWateringMessage = nextWaterDetails.message {
                        DetailHStack(key: "Message", value: nextWateringMessage)
                    }
                }

            }

            Section("Watering History (5 events in last 7 days)") {
                if let waterHistory = zone.history {
                    DetailHStack(key: "Count", value: String(waterHistory.count))
                    DetailHStack(key: "Average", value: waterHistory.average)
                    DetailHStack(key: "Total", value: waterHistory.total)
                    if let history = waterHistory.history {
                        if history.count > 0 {
                            let lastWateredTime = history[0].recordTime
                            DetailHStack(key: "Last Watered", value: lastWateredTime.formattedWithTime)
                            NavigationLink(destination: ZoneWaterHistoryView(waterHistory: waterHistory)) {
                                Text("See Full History")
                            }
                        }
                    }
                }
            }
        }
        .listStyle(InsetGroupedListStyle())
        .navigationTitle(zone.name)
        .onAppear { modelData.fetchZoneWaterHistory(zone: zone, range: "168h", limit: 5) }
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { showingEditZone.toggle() }) {
                    Text("Edit").accessibilityLabel("EditZone")
                }
            }
        }
        .sheet(isPresented: $showingEditZone) {
            EditZoneHost(garden: garden, zone: zone)
                .environmentObject(modelData)
//                .onDisappear { modelData.fetchGardens() }
        }
    }
}

struct ActionButtonStyle: ButtonStyle {
    var bgColor: Color
    
    func makeBody(configuration: Self.Configuration) -> some View {
        configuration.label
            .padding()
            .background(bgColor)
            .foregroundColor(.white)
            .cornerRadius(4)
            .padding(10)
            .scaleEffect(configuration.isPressed ? 0.90 : 1)
    }
}
