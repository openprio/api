import argparse
import json
import requests
import csv
import os

def generate_vehicle_pre_registrations(data_owner_code, ranges, output_json, output_csv):
    api_url = 'https://api.openprio.nl/auth/generate_vehicle_pre_registrations'

    registrations = []

    for range_str in ranges:
        start, end = map(int, range_str.split(','))
        for vehicle_number in range(start, end + 1):
            registration_data = {
                "data_owner_code": data_owner_code,
                "vehicle_number": str(vehicle_number)
            }
            registrations.append(registration_data)

    headers = {'Content-Type': 'application/json', 'apikey': os.getenv("APIKEY")}
    response = requests.post(api_url, json=registrations, headers=headers)

    if response.status_code == 200:

        # Save response in JSON format
        with open(output_json, 'w') as json_file:
            json.dump(response.json(), json_file, indent=2)
        print(f"Response saved in JSON file: {output_json}")

        # Save response in CSV format
        csv_data = response.json()
        csv_headers = ["data_owner_code", "vehicle_number", "token", "created_at"]
        with open(output_csv, 'w', newline='') as csv_file:
            csv_writer = csv.DictWriter(csv_file, fieldnames=csv_headers)
            csv_writer.writeheader()
            csv_writer.writerows(csv_data)
        print(f"Response saved in CSV file: {output_csv}")

    else:
        print(f"API call failed with status code: {response.status_code}")
        print(response.text)

def main():
    parser = argparse.ArgumentParser(description='Generate vehicle pre-registrations using OpenPrio API.')
    parser.add_argument('data_owner_code', help='Data owner code for the vehicles')
    parser.add_argument('vehicle_number_ranges', nargs="+", help='Comma-separated ranges for vehicle numbers (e.g., 3100,3137 5000,5005)')
    parser.add_argument('--output_json', default='pre_registrations.json', help='Output file for JSON response (default: pre_registrations.json)')
    parser.add_argument('--output_csv', default='pre_registrations.csv', help='Output file for CSV response (default: pre_registrations.csv)')

    args = parser.parse_args()

    generate_vehicle_pre_registrations(args.data_owner_code, args.vehicle_number_ranges, args.output_json, args.output_csv)

if __name__ == "__main__":
    main()
