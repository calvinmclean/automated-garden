//
//  GardensList.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/21.
//

import Foundation
import SwiftUI

struct GardensList: View {
    @EnvironmentObject var modelData: ModelData
    @State private var showingSettings = false
    @State private var showingAddGarden = false
    @State private var sortBy = Garden.SortBy.name
    @State private var reverseSort = false
    
    func toggleSortBy(buttonPressed: Garden.SortBy) {
        switch sortBy {
        case buttonPressed: // If buttonPressed is the same as current selection, reset
            sortBy = Garden.SortBy.name
        default:
            sortBy = buttonPressed
        }
    }
    
    var filteredGardens: [Garden] {
        let sortedGardens = modelData.gardens
            .filter { !$0.isEndDated() }
            .sorted {
                $0.isLessThan(other: $1, sortBy: sortBy)
            }
        return reverseSort ? sortedGardens.reversed() : sortedGardens
    }
    
    var endDatedGardens: [Garden] {
        let sortedGardens = modelData.gardens
            .filter { $0.isEndDated() }
            .sorted {
                $0.isLessThan(other: $1, sortBy: sortBy)
            }
        return reverseSort ? sortedGardens.reversed() : sortedGardens
    }
    
    func endDateGarden(at indexSet: IndexSet) {
        for index in indexSet {
            modelData.gardenResource().endDateGarden(garden: filteredGardens[index])
            modelData.fetchGardens()
        }
    }
    
    func permanentDeleteGarden(at indexSet: IndexSet) {
        for index in indexSet {
            modelData.gardenResource().endDateGarden(garden: endDatedGardens[index])
            modelData.fetchGardens()
        }
    }
    
    var body: some View {
        NavigationView {
            List {
                Section(header: Text("Active Gardens")) {
                    ForEach(filteredGardens) { garden in
                        NavigationLink(destination: GardenDetail(garden: garden)) {
                            GardenRow(garden: garden)
                        }
                    }
                    .onDelete(perform: self.endDateGarden)
                }
                Section(header: Text("End Dated Gardens")) {
                    ForEach(endDatedGardens) { garden in
                        NavigationLink(destination: GardenDetail(garden: garden)) {
                            GardenRow(garden: garden)
                        }
                    }
                    .onDelete(perform: self.permanentDeleteGarden)
                }
            }
            .refreshable{ modelData.fetchGardens() }
            .listStyle(InsetGroupedListStyle())
            .navigationTitle("Gardens")
            .onAppear { modelData.fetchGardens() }
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { showingSettings.toggle() }) {
                        Image(systemName: "gear")
                            .accessibilityLabel("Settings")
                    }
                }
                ToolbarItemGroup(placement: .bottomBar) {
                    Button(action: { toggleSortBy(buttonPressed: .name) }) {
                        Image(systemName: (sortBy == .name ? "a.circle.fill" : "a.circle"))
                            .accessibilityLabel("Toggle Sort By Name")
                    }
                    Button(action: { toggleSortBy(buttonPressed: .createdAt) }) {
                        Image(systemName: (sortBy == .createdAt ? "calendar.circle.fill" : "calendar.circle"))
                            .accessibilityLabel("Toggle Sort By Start Date")
                    }
                    Divider()
                    Button(action: { reverseSort.toggle() }) {
                        Image(systemName: (reverseSort ? "arrow.up" : "arrow.down"))
                            .accessibilityLabel("Reverse Sort")
                    }
                    Spacer()
                    Button(action: { showingAddGarden.toggle() }) {
                        Image(systemName: "plus")
                            .accessibilityLabel("Add Garden")
                    }
                }
            }
            .sheet(isPresented: $showingSettings) {
                SettingsHost()
                    .environmentObject(modelData)
                    .onDisappear { modelData.fetchGardens() }
            }
            .sheet(isPresented: $showingAddGarden) {
                AddGardenHost()
                    .environmentObject(modelData)
                    .onDisappear { modelData.fetchGardens() }
            }
        }
    }
}

struct GardensList_Preview: PreviewProvider {
    static var previews: some View {
        GardensList()
        .environmentObject(ModelData())
        .background(Color.green)
    }
}
