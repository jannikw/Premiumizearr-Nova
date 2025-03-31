<script>
  import { CalculateAPIPath } from "../Utilities/web_root";
  import { DataTable, InlineLoading } from "carbon-components-svelte";


  export let headers = [];
  export let APIpath = "";
  export let updateTimeSeconds = 5;
  export let zebra = false;
  export let totalName = "";
  export let transform = (data) => data; // Default transform function

  let updating = false;
  let status = "";
  $: rows=[]
  $: console.log("Rows updated:", rows);
  $: statusIndicator = updating ? "active" : "finished";

  function UpdateFromAPI() {
    if (updating) {
      console.log("UpdateFromAPI skipped because updating is true");
      return;
    }
    updating = true;
    console.log("Fetching data from API:", APIpath); // Debugging log
    fetch(CalculateAPIPath(APIpath))
      .then((res) => res.json())
      .then((data) => {
        console.log("API response:", data);
        if (data.data && data.data.length > 0) {
          console.log("Calling transform function:", transform);
          rows = [...transform(data.data)]; // Use the transform function passed as a prop
        } else {
          console.warn("API returned empty data, keeping previous rows.");
        }
        console.log("Updated rows in APITable:", rows);
        status = data.status;
        updating = false;
      })
      .catch((err) => {
        console.error("Error fetching data:", err);
        updating = false;
      }).finally(() => {
        updating = false; // Always reset updating flag
      });
  }

  function safeLength(obj) {
    return obj ? Object.keys(obj).length : 0;
  }

  UpdateFromAPI();
  setInterval(() => {
    UpdateFromAPI();
  }, updateTimeSeconds * 1000);



</script>

<main>
  {#if totalName !== ""}
    <p>
      {totalName}
      {safeLength(rows)}
    </p>
  {/if}
  <p>
    <InlineLoading status={statusIndicator} description="Update status" />
  </p>
  <p>
    Message: {status}
  </p>
  <p>
    <DataTable sortable {headers} {rows} rowKey="name" />
  </p></main>