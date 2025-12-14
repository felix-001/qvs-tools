package html

import "fmt"

type Table struct {
	agGridJs string
	rowDatas string
	columns  string
}

func NewTable() *Table {
	return &Table{
		agGridJs: "https://cdn.jsdelivr.net/npm/ag-grid-community/dist/ag-grid-community.min.js",
	}
}

func (t *Table) SetRowDatas(rowDatas string) {
	t.rowDatas = rowDatas
}

func (t *Table) SetColumns(columns string) {
	t.columns = columns
}

func (t *Table) String() string {
	s := "<div margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;>"
	s += `<div id="myGrid1" style="width: 100%; height: 200px" class="ag-theme-alpine"></div>`
	s += fmt.Sprintf(`<script src="%s"></script>`, t.agGridJs)
	s += fmt.Sprintf(`
		<script>
			console.log('create table');
			var gridOptions = {
				rowData: %s, 
				columnDefs: %s,
				defaultColDef: {
                			sortable: true,
                			filter: true
            			 },
			};
			console.log('当前状态:', document.readyState);
			document.addEventListener('DOMContentLoaded', () => {
				console.log('DOMContentLoaded event');
	    	    		agGrid.createGrid(document.querySelector("#myGrid1"), gridOptions);
        	        });
		</script>`,
		t.rowDatas, t.columns)
	s += "</div>"
	return s
}
