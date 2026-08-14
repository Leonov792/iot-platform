import SwiftUI

// MARK: - Конструктор автоматизаций (простые блоки «Если / То»)

struct AutomationRuleDraft: Identifiable {
    let id = UUID()
    var name = ""
    var deviceID = ""
    var field = ""
    var op = "gt"
    var threshold: Double = 0
    var actionDeviceID = ""
    var relay = ""
    var value = true
}

struct AutomationBuilderView: View {
    var automationURL: String = "http://localhost:8091"
    var automationToken: String = "dev-rules-token"

    @State private var rules: [AutomationRuleDraft] = [AutomationRuleDraft()]
    @State private var saving = false
    @State private var message: String?

    private let ops = ["gt", "lt", "gte", "lte", "eq", "neq"]

    var body: some View {
        NavigationStack {
            List {
                ForEach($rules) { $rule in
                    RuleCardView(rule: $rule, ops: ops)
                }
                .onDelete { idx in
                    rules.remove(atOffsets: idx)
                    if rules.isEmpty { rules.append(AutomationRuleDraft()) }
                }

                Button("+ Добавить правило") {
                    rules.append(AutomationRuleDraft())
                }
            }
            .navigationTitle("Автоматизации")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Сохранить") { save() }.disabled(saving)
                }
            }
            .overlay(alignment: .bottom) {
                if let message {
                    Text(message)
                        .font(.footnote)
                        .padding(8)
                        .background(.thinMaterial, in: Capsule())
                        .padding(.bottom, 8)
                }
            }
        }
    }

    private func save() {
        let payload = rules.compactMap { r -> [String: Any]? in
            guard !r.deviceID.isEmpty, !r.field.isEmpty else { return nil }
            return [
                "id": r.name.isEmpty ? UUID().uuidString : r.name,
                "name": r.name,
                "condition": ["device_id": r.deviceID, "field": r.field, "op": r.op, "value": r.threshold],
                "actions": [
                    ["type": "modbus_write", "device_id": r.actionDeviceID, "relay": r.relay, "value": r.value]
                ],
                "cooldown": "1m"
            ]
        }

        saving = true
        var req = URLRequest(url: URL(string: automationURL + "/v1/rules")!)
        req.httpMethod = "PUT"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue(automationToken, forHTTPHeaderField: "X-Automation-Token")
        req.httpBody = try? JSONSerialization.data(withJSONObject: payload)

        URLSession.shared.dataTask(with: req) { data, resp, err in
            DispatchQueue.main.async {
                saving = false
                if err == nil, let http = resp as? HTTPURLResponse, http.statusCode == 200 {
                    message = "Сохранено"
                } else {
                    message = "Ошибка сохранения"
                }
            }
        }.resume()
    }
}

struct RuleCardView: View {
    @Binding var rule: AutomationRuleDraft
    let ops: [String]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField("Название правила", text: $rule.name)

            HStack {
                Text("ЕСЛИ")
                TextField("device_id", text: $rule.deviceID)
                TextField("поле", text: $rule.field)
                Picker("", selection: $rule.op) {
                    ForEach(ops, id: \.self) { Text($0) }
                }
                .frame(width: 70)
                TextField("значение", value: $rule.threshold, format: .number)
                    .keyboardType(.decimalPad)
            }

            HStack {
                Text("ТО")
                TextField("device_id", text: $rule.actionDeviceID)
                TextField("реле", text: $rule.relay)
                Toggle("вкл", isOn: $rule.value)
            }
        }
        .font(.footnote)
        .textFieldStyle(.roundedBorder)
        .padding(.vertical, 4)
    }
}
