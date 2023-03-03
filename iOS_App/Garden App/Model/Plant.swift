//
//  Plant.swift
//  Garden App
//
//  Created by Calvin McLean on 6/11/21.
//

import Foundation

class Plant: Identifiable, Codable {
    var name: String = ""
    var details: PlantDetails = PlantDetails()
    var id: String = ""
    var createdAt: Date = Date()
    var endDate: Date? = nil
    var links: Array<Link> = []
    
    static let `default` = Plant()
    
    enum SortBy {
        case name
        case createdAt
    }
    
    static var decoder: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = JSONDecoder.DateDecodingStrategy.customISO8601
        return decoder
    }
    
    static var encoder: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = JSONEncoder.DateEncodingStrategy.customISO8601
        return encoder
    }
    
    func getLink(rel: String) -> String? {
        return self.links.first { l in
            l.rel == rel
        }?.href
    }
    
    func isEndDated() -> Bool {
        if let endDate = self.endDate {
            return endDate <= Date()
        }
        return false
    }
    
    func isLessThan(other: Plant, sortBy: SortBy) -> Bool {
        switch (self.isEndDated(), other.isEndDated(), sortBy) {
        case (true, true, .name), (false, false, .name):
            return self.name < other.name
        case (true, true, .createdAt), (false, false, .createdAt):
            return self.createdAt < other.createdAt
        case (true, false, _):
            return false
        case (false, true, _):
            return true
        }
    }
    
    enum CodingKeys: String, CodingKey {
        case name, id, details, links
        case createdAt = "created_at"
        case endDate = "end_date"
    }
}

struct PlantDetails: Codable {
    var description: String? = nil
    var notes: String? = nil
    var timeToHarvest: String? = nil
    var count: Int? = nil
    
    enum CodingKeys: String, CodingKey {
        case description, notes, count
        case timeToHarvest = "time_to_harvest"
    }
}

struct ListOfPlants: Codable {
    let plants: [Plant]
}

struct Plants<T: Codable>: Codable {
    let plants: [T]
}
