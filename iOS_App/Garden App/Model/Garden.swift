//
//  Garden.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/21.
//

import Foundation

class Garden: Identifiable, Codable {
    var name: String = ""
    var topicPrefix: String = ""
    var id: String = ""
    var maxZones: Int = 0
    var zones: Link = Link()
    var plants: Link = Link()
    var createdAt: Date = Date()
    var endDate: Date? = nil
    var lightSchedule: LightSchedule? = nil
    var nextLightAction: NextLightAction? = nil
    var health: GardenHealth? = nil
    var numPlants: Int = 0
    var numZones: Int = 0
    var links: Array<Link> = []
    
    static let `default` = Garden()

    enum SortBy {
        case name
        case createdAt
        case nextLightAction
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
    
    func isLessThan(other: Garden, sortBy: SortBy) -> Bool {
        switch (self.isEndDated(), other.isEndDated(), sortBy) {
        case (true, true, .name), (false, false, .name):
            return self.name < other.name
        case (true, true, .createdAt), (false, false, .createdAt):
            return self.createdAt < other.createdAt
        case (true, true, .nextLightAction), (false, false, .nextLightAction):
            return self.nextLightAction?.time ?? Date() < other.nextLightAction?.time ?? Date()
        case (true, false, _):
            return false
        case (false, true, _):
            return true
        }
    }
    
    enum CodingKeys: String, CodingKey {
        case name, id, zones, plants, health, links
        case topicPrefix = "topic_prefix"
        case maxZones = "max_zones"
        case createdAt = "created_at"
        case endDate = "end_date"
        case lightSchedule = "light_schedule"
        case nextLightAction = "next_light_action"
        case numPlants = "num_plants"
        case numZones = "num_zones"
    }
}

struct LightSchedule: Codable {
    var duration: String = ""
    var startTime: String = ""
    
    enum CodingKeys: String, CodingKey {
        case duration
        case startTime = "start_time"
    }
}

struct NextLightAction: Codable {
    var time: Date = Date()
    var state: String = ""
    
    enum CodingKeys: String, CodingKey {
        case time, state
    }
}

struct ListOfGardens: Codable {
    let gardens: [Garden]
}

struct Gardens<T: Codable>: Codable {
    let gardens: [T]
}

struct GardenHealth: Codable {
    var status: String = "N/A"
    var details: String = ""
    var lastContact: Date? = nil
    
    enum CodingKeys: String, CodingKey {
        case status, details
        case lastContact = "last_contact"
    }
}
