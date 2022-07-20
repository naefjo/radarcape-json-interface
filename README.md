# radarcape-json-interface
This repo implements an interface to the radarcape Jetvision which downloads and saves incoming data for specified aircraft types. 
This interface was developed as part of my civilian service at the Swiss Federal Laboratories for Materials Science and Technology (EMPA). 

The application interfaces with the Radarcape using HTTP and downloads the decoded data which the Radarcape provides in JSON format. Subsequently, the data is filtered
according to the specified aircraft types and then stored in CSV files per day per aircraft type.
