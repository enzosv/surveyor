
var token;
async function initialize(success) {
    if (!firebase.apps.length) {
        const configRequest = await fetch(window.location.origin+'/firebase_web.json')
        const firebaseConfig = await configRequest.json()
        console.log(firebase.apps.length)
        const app = firebase.initializeApp(firebaseConfig);
    }
    
    
    firebase.auth().onAuthStateChanged((user) => {
        if(!user){
            if(window.location.href != window.location.origin + "/") {
                console.log("redirect")
                window.location.replace("/");
                return
            }
            firebaseSignIn()
            return
        }
        if(window.location.href == window.location.origin + "/") {
            window.location.replace("/questions");
            return
        }
        user.getIdToken(true).then(function(idToken) {
            token = idToken
            signin(idToken)
            success(idToken)
            
        }).catch(function(error) {
            // if(window.location.href != window.location.origin + "/") {
            //     console.log("redirect")
            //     window.location.replace("/");
            //     return
            // }
            // console.error(error)
            // firebaseSignIn()
            // return
        });
    });
}

function signout() {
    firebase.auth().signOut().then(() => {
        window.location.replace("/");
      }).catch((error) => {
          console.error(error)
      });
}

async function signin(idToken) {
    let request = await fetch("/signin", {
        method: 'POST',
        headers: {
            'Accept': 'application/json',
            'Authorization': 'Token ' + idToken
        }
    })
    let response = await request.json()
}

function firebaseSignIn() {
    var ui = new firebaseui.auth.AuthUI(firebase.auth());
    var uiConfig = {
        callbacks: {
            signInSuccessWithAuthResult: function (authResult, redirectUrl) {
                // User successfully signed in.
                // Return type determines whether we continue the redirect automatically
                // or whether we leave that to developer to handle.
                return true;
            }
        },
        // Will use popup for IDP Providers sign-in flow instead of the default, redirect.
        signInFlow: 'popup',
        signInSuccessUrl: '/questions',
        signInOptions: [
            {
                provider: firebase.auth.GoogleAuthProvider.PROVIDER_ID
            },
            {
                provider: firebase.auth.EmailAuthProvider.PROVIDER_ID
            }
        ],
        // Terms of service url.
        tosUrl: '<your-tos-url>',
        // Privacy policy url.
        privacyPolicyUrl: '<your-privacy-policy-url>'
    };
    firebase.auth().setPersistence(firebase.auth.Auth.Persistence.LOCAL)
    .then(() => {
        ui.start('#firebaseui-auth-container', uiConfig);
    })
    .catch((error)=>{
        console.error(error)
    })
}