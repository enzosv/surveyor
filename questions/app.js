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
    let questions = await response.json();
    let container = document.getElementById("questions")
    questions.forEach(element => {
        question_ids.push(element.question_id)
        let html = `<h4 class="pt-5">${element.statement}<h4>`
        for(var i=1;i<8;i++){
            html += `<input type="radio" required class="btn-check btn-block" name="${element.question_id}" id="${element.question_id}:${i}" value="${i}" ${element.response==i? "checked" : ""}>
            <label class="btn btn-outline-success" for="${element.question_id}:${i}">${i}</label>`
        }
        let div = document.createElement("div")
        div.innerHTML = html
        container.prepend(div)
    });
}

async function answer() {
    var data = []
    question_ids.forEach(id => {
        let answer = document.querySelector(`input[name="${id}"]:checked`).value;
        data.push({"question_id": id, "answer":parseInt(answer)})
    })
    await fetch("/daily", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token '+ token
        },
        body: JSON.stringify(data)
    }).then((response) => {
        if (response.status >= 400 && response.status < 600) {
          throw new Error("Bad response from server");
        }
        return response;
    }).then((returnedResponse) => {
        console.log(returnedResponse)
       // Your response to manipulate
       alert("Thank you. Your response has been recorded.")
    }).catch((error) => {
      // Your error is here!
      console.log(error)
      alert("Something went wrong. Please inform an administrator.")
    });
}

initialize( function(idToken){
    main(idToken)
})