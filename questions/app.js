var question_ids = []
async function main(idToken){
    question_ids = []
    let response = await fetch("/daily", {
        method: 'GET',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+idToken
        }
    })
    console.log(idToken)
    let questions = await response.json();
    let container = document.getElementById("questions")
    questions.forEach(element => {
        question_ids.push(element.question_id)
        let html = `
        <h4 class="pt-5">${element.statement}<h4>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:1" value="1">
        <label class="btn btn-outline-success" for="${element.question_id}:1">1</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:2" value="2">
        <label class="btn btn-outline-success" for="${element.question_id}:2">2</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:3" value="3">
        <label class="btn btn-outline-success" for="${element.question_id}:3">3</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:4" value="4">
        <label class="btn btn-outline-success" for="${element.question_id}:4">4</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:5" value="5">
        <label class="btn btn-outline-success" for="${element.question_id}:5">5</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:6" value="6">
        <label class="btn btn-outline-success" for="${element.question_id}:6">6</label>
        <input type="radio" required class="btn-check" name="${element.question_id}" id="${element.question_id}:7" value="7">
        <label class="btn btn-outline-success" for="${element.question_id}:7">7</label>
        `
        let div = document.createElement("div")
        div.innerHTML = html
        container.prepend(div)
    });
}

function answer() {
    var data = []
    question_ids.forEach(id => {
        let answer = document.querySelector(`input[name="${id}"]:checked`).value;
        data.push({"question_id": id, "answer":parseInt(answer)})
    })
    console.log(data, token)
    fetch("/daily", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+ token
        },
        body: JSON.stringify(data)
    })
}

initialize( function(idToken){
    main(idToken)
})