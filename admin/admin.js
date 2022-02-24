$(document).ready(function () {
    $('#construct_slug').on('keyup', function () {
        if (!$.fn.dataTable.isDataTable('#constructs')) {
            return
        }
        // TODO: ignore filter
        var table = $('#constructs').DataTable()
        table.search(this.value).draw();
    });
    $('#facet_name').on('keyup', function () {
        if (!$.fn.dataTable.isDataTable('#facets')) {
            return
        }
        // TODO: ignore filter
        var table = $('#facets').DataTable()
        table.search(this.value).draw();
    });
    $('#statement').on('keyup', function () {
        if (!$.fn.dataTable.isDataTable('#questions')) {
            return
        }

        // TODO: ignore filter
        var table = $('#questions').DataTable()
        table.search(this.value).draw();
    });

    
});

initialize(function (idToken) {
    fetchConstructs()
    fetchFacets()
    fetchQuestions()
}, function (err) {
    console.error(err)
})

async function setConstruct() {
    let name = $("#construct_name").val()
    let data = {
        "slug": name.toLowerCase().replace(/ /g, "-"),
        "name": name
    }
    let response = await fetch("/admin/constructs", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        },
        body: JSON.stringify(data)
    })
    console.log(await response.json())
    fetchConstructs()
}

async function createFacet() {
    let data = {
        "construct_id": $("#construct_id").val(),
        "name": $("#facet_name").val()
    }
    console.log(data)
    await fetch("/admin/facets", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        },
        body: JSON.stringify(data)
    })
    fetchFacets()
}

async function createQuestion() {
    let data = {
        "facet_id": parseInt($("#facet_id").val()),
        "statement": $("#statement").val(),
        "is_reverse": document.getElementById("is_reverse").checked,
    }
    console.log(data)
    await fetch("/admin/questions", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        },
        body: JSON.stringify(data)
    })
    fetchQuestions()
}


function fetchConstructs() {
    $("#construct_id").empty();
    $("#construct_id").append(new Option("Select Construct"));
    var table
    if (!$.fn.dataTable.isDataTable('#constructs')) {
        table = $('#constructs').DataTable({
            "ajax": {
                "url": 'constructs',
                "type": 'GET',
                "beforeSend": function (xhr) {
                    xhr.setRequestHeader('Authorization', "Token " + token);
                }
            },
            "columns": [
                { "data": "name" },
                {
                    "data": "created_at",
                    render: function (data, type, row) {
                        return new Date(data).toLocaleDateString('en-us', date_options);
                    },
                },
                {
                    "data": null,
                    "defaultContent": '<button class="btn btn-danger btn-sm">Delete</button>',
                    "orderable": false
                }
            ],
            scorller: false,
        });
        table.on('xhr', function () {
            var data = table.ajax.json().data;
            if (data == undefined || data.length < 1) {
                return
            }
            let select = document.getElementById("construct_id")
            data.forEach(element => {
                select.append(new Option(element.slug, element.construct_id));
            })
        });
        $('#constructs tbody').on('click', 'button', function () {
            let data = table.row($(this).parents('tr')).data();
            console.log(data.construct_id)
            if (this.innerHTML == "Delete") {
                deleteConstruct(data.construct_id, table)
            } else if (this.innerHTML == "Edit") {

            }
        });
    } else {
        table = $('#constructs').DataTable();
        table.ajax.reload()
    }
}

const date_options = { year: "numeric", month: "short", day: "numeric" }

function fetchFacets() {
    $("#facet_id").empty();
    $("#facet_id").append(new Option("Select Facet"));
    var table
    if (!$.fn.dataTable.isDataTable('#facets')) {
        table = $('#facets').DataTable({
            "ajax": {
                "url": 'facets',
                "type": 'GET',
                "beforeSend": function (xhr) {
                    xhr.setRequestHeader('Authorization', "Token " + token)
                }
            },
            "columns": [
                { "data": "construct" },
                { "data": "name" },
                {
                    "data": "created_at",
                    render: function (data, type, row) {
                        return new Date(data).toLocaleDateString('en-us', date_options);
                    },
                },
                {
                    "data": null,
                    "defaultContent": '<button class="btn btn-danger btn-sm">Delete</button>',
                    "orderable": false
                }
            ]

        });
        table.on('xhr', function () {
            var data = table.ajax.json().data;
            if (data == undefined || data.length < 1) {
                return
            }
            let select = $("#facet_id")
            data.forEach(element => {
                select.append(new Option(element.name, element.facet_id));
            })
        });
        $('#facets tbody').on('click', 'button', function () {
            let data = table.row($(this).parents('tr')).data();
            console.log(data.facet_id)
            if (this.innerHTML == "Delete") {
                deleteFacet(data.facet_id, table)
            } else if (this.innerHTML == "Edit") {

            }
        });
    } else {
        table = $('#facets').DataTable();
        table.ajax.reload()
    }

}

async function deleteConstruct(facet_id, table) {
    await fetch("/admin/constructs/" + facet_id, {
        method: 'DELETE',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        }
    })
    table.ajax.reload()
}

async function deleteFacet(facet_id, table) {
    await fetch("/admin/facets/" + facet_id, {
        method: 'DELETE',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        }
    })
    table.ajax.reload()
}

async function deleteQuestion(facet_id, table) {
    await fetch("/admin/questions/" + facet_id, {
        method: 'DELETE',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + token
        }
    })
    table.ajax.reload()
}

function fetchQuestions() {
    var table
    if (!$.fn.dataTable.isDataTable('#questions')) {
        table = $('#questions').DataTable({
            "ajax": {
                "url": 'questions',
                "type": 'GET',
                "beforeSend": function (xhr) {
                    xhr.setRequestHeader('Authorization', "Token " + token)
                }
            },
            "columns": [
                { "data": "facet" },
                { "data": "statement" },
                {
                    "data": "is_reverse",
                    render: function (data, type, row) {
                        return `<div class="form-check form-switch">
                        <center><input class="form-check-input" type="checkbox" disabled ${data ? "checked" : ""}></center>
                        </div>`;
                    },
                },
                {
                    "data": "created_at",
                    render: function (data, type, row) {
                        return new Date(data).toLocaleDateString('en-us', date_options);
                    },
                },
                {
                    "data": null,
                    "defaultContent": '<button class="btn btn-danger btn-sm">Delete</button>',
                    "orderable": false
                }
            ]
        });
        $('#questions tbody').on('click', 'button', function () {
            let data = table.row($(this).parents('tr')).data();
            console.log(data.question_id)
            if (this.innerHTML == "Delete") {
                deleteQuestion(data.question_id, table)
            } else if (this.innerHTML == "Edit") {

            }
        });
    } else {
        table = $('#questions').DataTable();
        table.ajax.reload()
    }
}