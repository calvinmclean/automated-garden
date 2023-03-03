//
//  GardenResource.swift
//  Garden App
//
//  Created by Calvin McLean on 11/23/21.
//

import Foundation

struct GardenAction: Codable {
    var light: LightAction? = nil
    var stop: StopAction? = nil
}

struct StopAction: Codable {
    var all: Bool = false
}

struct LightAction: Codable {
    var state: String = ""
    var forDuration: String?
    
    enum CodingKeys: String, CodingKey {
        case state
        case forDuration = "for_duration"
    }
}

struct CreateGardenRequest: Codable {
    var name: String = ""
    var topicPrefix: String = ""
    var maxZones: Int = 1
    var lightSchedule: LightSchedule? = nil
    
    enum CodingKeys: String, CodingKey {
        case name
        case topicPrefix = "topic_prefix"
        case maxZones = "max_zones"
        case lightSchedule = "light_schedule"
    }
}

class GardenResource {
    var url: String
    
    init(url: String) {
        self.url = url
    }
    
    func getGardens(showEndDated: Bool = false, withCompletion completion: @escaping ([Garden]?) -> Void) {
        guard let url = URL(string: "\(self.url)/gardens?end_dated=\(showEndDated)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        let task = URLSession.shared.dataTask(with: request) { (data, response, error) in
            if let data = data {
                do {
                    let gardens = try Garden.decoder.decode(ListOfGardens.self, from: data)
                    DispatchQueue.main.async { completion(gardens.gardens) }
                } catch let jsonErr {
                    print("JSON Error: \(jsonErr)")
                    //                    print("DATA: \(String(data: data, encoding: String.Encoding.utf8))")
                    DispatchQueue.main.async { completion(nil) }
                }
            }
        }
        task.resume()
    }
    
    func endDateGarden(garden: Garden) {
        guard let path = garden.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func stopWatering(garden: Garden, all: Bool = false) {
        guard let path = garden.getLink(rel: "action") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let requestBody = GardenAction(stop: StopAction(all: all))
        let jsonData = try! JSONEncoder().encode(requestBody)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func restoreGarden(garden: Garden) {
        guard let path = garden.getLink(rel: "self") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        let data: Dictionary<String, Date?> = ["end_date": nil]
        let jsonData = try! JSONEncoder().encode(data)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func toggleLight(garden: Garden, state: String = "") {
        guard let path = garden.getLink(rel: "action") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let requestBody = GardenAction(light: LightAction(state: state))
        let jsonData = try! JSONEncoder().encode(requestBody)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
    
    func delayLight(garden: Garden, minutes: Int = 30, state: String = "OFF") {
        guard let path = garden.getLink(rel: "action") else { fatalError() }
        guard let url = URL(string: "\(self.url)\(path)") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let requestBody = GardenAction(light: LightAction(state: state, forDuration: "\(minutes)m"))
        let jsonData = try! JSONEncoder().encode(requestBody)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }

    func createGarden(garden: CreateGardenRequest) {
        guard let url = URL(string: "\(self.url)/gardens") else { fatalError() }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        let jsonData = try! JSONEncoder().encode(garden)
        request.httpBody = jsonData
        let task = URLSession.shared.dataTask(with: request)
        task.resume()
    }
}
