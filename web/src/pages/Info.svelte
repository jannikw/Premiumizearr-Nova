<script>
  import APITable from "../components/APITable.svelte";
  import { Row, Column } from "carbon-components-svelte";

  let dlSpeed = 0;

  function parseDLSpeedFromMessage(m) {
    if (m == "Loading..." || m == undefined) return 0;
    if (m == "too many missing articles") return 0;
    
    let speed = m.split(" ")[0];
    speed = speed.replace(",", "");
    let unit = m.split(" ")[1];
    if (Number.isNaN(speed)) {
      console.log("Speed is not a number: ", speed);
      console.log("Message: ", message);
      return 0;
    }
    if (unit === undefined || unit === null || unit == "") {
      console.log("Unit undefined in : " + m);
      return 0;
    } else {
      try {
        unit = unit.toUpperCase();
      } catch (error) {
        return 0;
      }
      unit = unit.replace("/", "");
      unit = unit.substring(0, 2);
      switch (unit) {
        case "KB":
          return Number(speed) * 1024;
        case "MB":
          return Number(speed) * 1024 * 1024;
        case "GB":
          return Number(speed) * 1024 * 1024 * 1024;
        default:
          console.log("Unknown unit: " + unit + " in message '" + m + "'");
          return 0;
      }
    }
  }

  function HumanReadableSpeed(bytes) {
    if (bytes < 1024) {
      return bytes + " B/s";
    } else if (bytes < 1024 * 1024) {
      return (bytes / 1024).toFixed(2) + " KB/s";
    } else if (bytes < 1024 * 1024 * 1024) {
      return (bytes / 1024 / 1024).toFixed(2) + " MB/s";
    } else {
      return (bytes / 1024 / 1024 / 1024).toFixed(2) + " GB/s";
    }
  }

  function dataToRowsDownload(data) {
    console.log("Transforming data:", data); // Log the input data
    if (!data) return [];

    // Filter out rows with "0% Complete (0 B)" or similar variations
    const filteredData = data.filter(d => {
      console.log("Progress field values:", d.progress); // Log each progress field
      return !d.progress.includes("(0 B)");
    });

    const transformed = filteredData.map((d, index) => ({
      id: d.id || index, // Use `d.id` if available, otherwise fallback to the index
      added: d.added,
      name: d.name,
      progress: d.progress,
      speed: d.speed,
    }));

    console.log("Transformed Info rows:", transformed); // Log the transformed rows
    return transformed;
  }

  function dataToRows(data) {
      if (!data) return [];

      let dlSpeed = 0;

      return data.map(d => {
          let speed = parseDLSpeedFromMessage(d.message);
          if (!Number.isNaN(speed)) dlSpeed += speed;

          return {
              name: d.name,
              status: d.status,
              progress: (d.progress * 100).toFixed(0) + "%",
              message: d.message,
          };
      });
  }

</script>

<main>
    <Row>
      <Column md={4} >
        <h3>Blackhole</h3>
        <APITable
          headers={[
            { key: "id", value: "Pos" },
            { key: "name", value: "Name", sort: false },
          ]}
          APIpath="api/blackhole"
          zebra={true}
          totalName="In Queue: "
        />
      </Column>
      <Column md={4} >
        <h3>Downloads</h3>
        <APITable
          headers={[
            { key: "added", value: "Added" },
            { key: "name", value: "Name" },
            { key: "progress", value: "Progress" },
            { key: "speed", value: "Speed" },
          ]}
          updateTimeSeconds={2}
          APIpath="api/downloads"
          zebra={true}
          totalName="Downloading: "
          transform={dataToRowsDownload}
        />
      </Column>
    </Row>
    <Row>
      <Column>
        <h3>Transfers</h3>
        <p>Download Speed: {HumanReadableSpeed(dlSpeed)}</p>
        <APITable
          headers={[
            { key: "name", value: "Name" },
            { key: "status", value: "Status" },
            { key: "progress", value: "Progress" },
            { key: "message", value: "Message", sort: false },
          ]}
          APIpath="api/transfers"
          zebra={true}
          transform={dataToRows}
        />
      </Column>
    </Row>
</main>
